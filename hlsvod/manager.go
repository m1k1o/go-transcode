package hlsvod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// how long can it take for transcode to be ready
const readyTimeout = 80 * time.Second

// how long can it take for transcode to return first data
const transcodeTimeout = 10 * time.Second

type ManagerCtx struct {
	logger zerolog.Logger
	config Config

	segmentLength float64
	segmentOffset float64

	ready     bool
	readyMu   sync.RWMutex
	readyChan chan struct{}

	metadata    *ProbeMediaData
	playlist    string    // m3u8 playlist string
	breakpoints []float64 // list of breakpoints for segments

	segments   map[int]string // map of segments and their filename
	segmentsMu sync.RWMutex

	segmentWait   map[int]chan struct{} // map of segments and signaling channel for finished transcoding
	segmentWaitMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

func New(config Config) *ManagerCtx {
	ctx, cancel := context.WithCancel(context.Background())
	return &ManagerCtx{
		logger: log.With().Str("module", "hlsvod").Str("submodule", "manager").Logger(),
		config: config,

		segmentLength: 3.50,
		segmentOffset: 1.25,

		ctx:    ctx,
		cancel: cancel,
	}
}

//
// ready
//

func (m *ManagerCtx) readyReset() {
	m.readyMu.Lock()
	defer m.readyMu.Unlock()

	m.ready = false
	m.readyChan = make(chan struct{})
}

func (m *ManagerCtx) readyDone() {
	m.readyMu.Lock()
	defer m.readyMu.Unlock()

	m.ready = true
	if m.readyChan != nil {
		close(m.readyChan)
	}
	m.readyChan = nil
}

func (m *ManagerCtx) isReady() bool {
	m.readyMu.RLock()
	defer m.readyMu.RUnlock()

	return m.ready
}

func (m *ManagerCtx) waitForReady() chan struct{} {
	m.readyMu.RLock()
	defer m.readyMu.RUnlock()

	return m.readyChan
}

//
// metadata
//

// fetch metadata using ffprobe
func (m *ManagerCtx) fetchMetadata() (err error) {
	start := time.Now()
	log.Info().Msg("fetching metadata")

	// start ffprobe to get metadata about current media
	m.metadata, err = ProbeMedia(m.ctx, m.config.FFprobeBinary, m.config.MediaPath)
	if err != nil {
		return fmt.Errorf("unable probe media for metadata: %v", err)
	}

	// if media has video, use keyframes as reference for segments
	if m.metadata.Video != nil && m.metadata.Video.PktPtsTime == nil {
		// start ffprobe to get keyframes from video
		videoData, err := ProbeVideo(m.ctx, m.config.FFprobeBinary, m.config.MediaPath)
		if err != nil {
			return fmt.Errorf("unable probe video for keyframes: %v", err)
		}
		m.metadata.Video.PktPtsTime = videoData.PktPtsTime
	}

	elapsed := time.Since(start)
	log.Info().Interface("duration", elapsed).Msg("fetched metadata")
	return
}

// load metadata from cache or fetch them and cache
func (m *ManagerCtx) loadMetadata() error {
	// bypass cache if not enabled
	if !m.config.Cache {
		return m.fetchMetadata()
	}

	// try to get cached data
	data, err := m.getCacheData()
	if err == nil {
		// unmarshall cache data
		err := json.Unmarshal(data, &m.metadata)
		if err == nil {
			return nil
		}

		log.Err(err).Msg("cache unmarhalling returned error, replacing")
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Err(err).Msg("cache hit returned error, replacing")
	}

	// fetch fresh metadata from a file
	if err := m.fetchMetadata(); err != nil {
		return err
	}

	// marshall new metadata to bytes
	data, err = json.Marshal(m.metadata)
	if err != nil {
		return err
	}

	if m.config.CacheDir != "" {
		return m.saveGlobalCacheData(data)
	}

	return m.saveLocalCacheData(data)
}

func (m *ManagerCtx) getSegmentName(index int) string {
	return fmt.Sprintf("%s-%05d.ts", m.config.SegmentPrefix, index)
}

func (m *ManagerCtx) parseSegmentIndex(segmentName string) (int, bool) {
	regex := regexp.MustCompile(`^(.*)-([0-9]{5})\.ts$`)
	matches := regex.FindStringSubmatch(segmentName)

	if len(matches) != 3 || matches[1] != m.config.SegmentPrefix {
		return 0, false
	}

	indexStr := matches[2]
	index, err := strconv.Atoi(indexStr)
	if indexStr == "" || err != nil {
		return 0, false
	}

	return index, true
}

func (m *ManagerCtx) getPlaylist() string {
	// playlist prefix
	playlist := []string{
		"#EXTM3U",
		"#EXT-X-VERSION:4",
		"#EXT-X-PLAYLIST-TYPE:VOD",
		"#EXT-X-MEDIA-SEQUENCE:0",
		fmt.Sprintf("#EXT-X-TARGETDURATION:%.2f", m.segmentLength+m.segmentOffset),
	}

	// playlist segments
	for i := 1; i < len(m.breakpoints); i++ {
		playlist = append(playlist,
			fmt.Sprintf("#EXTINF:%.3f, no desc", m.breakpoints[i]-m.breakpoints[i-1]),
			m.getSegmentName(i),
		)
	}

	// playlist suffix
	playlist = append(playlist,
		"#EXT-X-ENDLIST",
	)

	// join with newlines
	return strings.Join(playlist, "\n")
}

func (m *ManagerCtx) initialize() {
	keyframes := []float64{}
	if m.metadata.Video != nil && m.metadata.Video.PktPtsTime != nil {
		keyframes = m.metadata.Video.PktPtsTime
	}

	// generate breakpoints from keyframes
	m.breakpoints = convertToSegments(keyframes, m.metadata.Duration, m.segmentLength, m.segmentOffset)

	// generate playlist
	m.playlist = m.getPlaylist()

	// prepare transcode matrix from breakpoints
	m.segments = map[int]string{}
	for i := 1; i < len(m.breakpoints); i++ {
		m.segments[i] = ""
	}

	// prepare segment wait map
	m.segmentWait = map[int]chan struct{}{}

	log.Info().
		Int("segments", len(m.segments)).
		Bool("video", m.metadata.Video != nil).
		Int("audios", len(m.metadata.Audio)).
		Str("duration", fmt.Sprintf("%v", m.metadata.Duration)).
		Msg("initialization completed")
}

//
// segments
//

func (m *ManagerCtx) addSegment(index int, segmentName string) {
	m.segmentsMu.Lock()
	defer m.segmentsMu.Unlock()

	m.segments[index] = segmentName
}

func (m *ManagerCtx) getSegment(index int) (segmentPath string, transcoded, ok bool) {
	var segmentName string

	m.segmentsMu.RLock()
	segmentName, ok = m.segments[index]
	m.segmentsMu.RUnlock()

	if !ok {
		return
	}

	segmentPath = path.Join(m.config.TranscodeDir, segmentName)
	transcoded = segmentName != ""
	return
}

func (m *ManagerCtx) clearAllSegments() {
	m.segmentsMu.Lock()
	defer m.segmentsMu.Unlock()

	for _, segmentName := range m.segments {
		if segmentName == "" {
			continue
		}

		segmentPath := path.Join(m.config.TranscodeDir, segmentName)
		if err := os.Remove(segmentPath); err != nil {
			log.Err(err).Str("path", segmentPath).Msg("error while removing file")
		}
	}
}

func (m *ManagerCtx) transcodeSegment(ctx context.Context, index int) (chan struct{}, error) {
	// TODO: Optimize for more segments.
	segmentTimes := m.breakpoints[index : index+2]
	log.Info().Int("offset", index).Interface("segments-times", segmentTimes).Msg("transcoding segment")

	data, err := TranscodeSegments(ctx, m.config.FFmpegBinary, TranscodeConfig{
		InputFilePath: m.config.MediaPath,
		OutputDirPath: m.config.TranscodeDir,
		SegmentPrefix: m.config.SegmentPrefix, // This does not need to match.

		VideoProfile: m.config.VideoProfile,
		AudioProfile: m.config.AudioProfile,

		SegmentOffset: index,
		SegmentTimes:  segmentTimes,
	})

	if err != nil {
		log.Err(err).Int("index", index).Msg("error occured while starting to transcode segment")
		return nil, err
	}

	response := make(chan struct{}, 1)
	m.segmentWait[index] = response

	go func() {
		log.Info().Int("index", index).Msg("transcode process started")

		for {
			segmentName, ok := <-data
			if !ok {
				log.Info().Int("index", index).Msg("transcode process finished")
				return
			}

			log.Info().
				Int("index", index).
				Str("segment", segmentName).
				Msg("transcode process returned a segment")

			// add transcoded segment name
			m.addSegment(index, segmentName)

			// notify waiting element, if exists
			if res, ok := m.segmentWait[index]; ok {
				close(res)
				delete(m.segmentWait, index)
			}

			// expect new segment to come
			index++
		}
	}()

	return response, nil
}

func (m *ManagerCtx) Start() (err error) {
	// create new executing context
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// initialize ready state
	m.readyReset()

	// initialize transcoder asynchronously
	go func() {
		if err := m.loadMetadata(); err != nil {
			log.Printf("%v\n", err)
			return
		}

		// initialization based on metadata
		m.initialize()

		// set ready state as done
		m.readyDone()
	}()

	// TODO: Cleanup process.

	return nil
}

func (m *ManagerCtx) Stop() {
	// reset ready state
	m.readyReset()

	// cancel current context
	m.cancel()

	// TODO: stop all transcoding processes

	// remove all transcoded segments
	m.clearAllSegments()
}

func (m *ManagerCtx) Cleanup() {
	// TODO: check what segments are really needed
	// TODO: stop transcoding processes that are not needed anymore
}

func (m *ManagerCtx) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	// ensure that transcode started
	if !m.isReady() {
		select {
		// waiting for transcode to be ready
		case <-m.waitForReady():
			// check if it started succesfully
			if !m.isReady() {
				m.logger.Warn().Msgf("playlist load failed")
				http.Error(w, "504 playlist not available", http.StatusInternalServerError)
				return
			}
		// when transcode stops before getting ready
		case <-m.ctx.Done():
			m.logger.Warn().Msg("playlist load failed because of shutdown")
			http.Error(w, "500 playlist not available", http.StatusInternalServerError)
			return
		case <-time.After(readyTimeout):
			m.logger.Warn().Msg("playlist load timeouted")
			http.Error(w, "504 playlist timeout", http.StatusGatewayTimeout)
			return
		}
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write([]byte(m.playlist))
}

func (m *ManagerCtx) ServeMedia(w http.ResponseWriter, r *http.Request) {
	// same of the requested segment is everything after last slash
	reqSegName := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

	// getting index from segment name
	index, ok := m.parseSegmentIndex(reqSegName)
	if !ok {
		http.Error(w, "400 bad media path", http.StatusBadRequest)
		return
	}

	// check if segment exists
	segmentPath, isTranscoded, ok := m.getSegment(index)
	if !ok {
		http.Error(w, "404 index not found", http.StatusNotFound)
		return
	}

	// check if segment is transcoded
	if !isTranscoded {
		// check if segment transcoding is already in progress
		segChan, ok := m.segmentWait[index]
		if !ok {
			var err error
			segChan, err = m.transcodeSegment(r.Context(), index)

			// if transcode proccess could not start
			if err != nil {
				m.logger.Err(err).Int("index", index).Msg("unable to transcode media")
				http.Error(w, "500 unable to transcode", http.StatusInternalServerError)
				return
			}
		}

		select {
		// waiting for new segment to be transcoded
		case <-segChan:
			// now segment should be available
			segmentPath, isTranscoded, ok = m.getSegment(index)
			if !ok || !isTranscoded {
				// this should never happen
				http.Error(w, "404 segment not found even after transcoding", http.StatusNotFound)
				return
			}
		// when transcode stops before getting ready
		case <-m.ctx.Done():
			m.logger.Warn().Msg("media transcode failed because of shutdown")
			http.Error(w, "500 media not available", http.StatusInternalServerError)
			return
		case <-time.After(transcodeTimeout):
			m.logger.Warn().Msg("media transcode timeouted")
			http.Error(w, "504 media timeout", http.StatusGatewayTimeout)
			return
		}
	}

	// build whole segment path
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		m.logger.Warn().Int("index", index).Str("path", segmentPath).Msg("media file not found")
		http.Error(w, "404 media not found", http.StatusNotFound)
		return
	}

	// return existing segment
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, segmentPath)
}
