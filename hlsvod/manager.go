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
	mu     sync.Mutex
	config Config

	segmentLength float64
	segmentOffset float64

	ready         bool
	onReadyChange chan struct{}

	events struct {
		onStart  func()
		onCmdLog func(message string)
		onStop   func(err error)
	}

	metadata    *ProbeMediaData
	playlist    string         // m3u8 playlist string
	segments    map[int]string // map of segments and their filename
	breakpoints []float64      // list of breakpoints for segments

	shutdown chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
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

	log.Info().
		Int("segments", len(m.segments)).
		Bool("video", m.metadata.Video != nil).
		Int("audios", len(m.metadata.Audio)).
		Str("duration", fmt.Sprintf("%v", m.metadata.Duration)).
		Msg("initialization completed")
}

// TODO: Optimize for more segments.
func (m *ManagerCtx) transcodeSegment(ctx context.Context, index int) (chan string, error) {
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

	response := make(chan string)
	go func() {
		log.Info().Int("index", index).Msg("transcode process started")

		for {
			segment, ok := <-data
			if !ok {
				log.Info().Int("index", index).Msg("transcode process finished")
				return
			}
			log.Info().Int("index", index).Str("segment", segment).Msg("transcode process returned a segment")
			response <- segment
		}
	}()

	return response, nil
}

func (m *ManagerCtx) Start() (err error) {
	if m.ready {
		return fmt.Errorf("already running")
	}

	m.mu.Lock()
	// initialize signaling channels
	m.shutdown = make(chan struct{})
	m.ready = false
	m.onReadyChange = make(chan struct{})
	m.mu.Unlock()

	// initialize transcoder asynchronously
	go func() {
		if err := m.loadMetadata(); err != nil {
			log.Printf("%v\n", err)
			return
		}

		// initialization based on metadata
		m.initialize()

		m.mu.Lock()
		// set video to ready state
		m.ready = true
		close(m.onReadyChange)
		m.mu.Unlock()
	}()

	// TODO: Cleanup process.

	return nil
}

func (m *ManagerCtx) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// stop all transcoding processes
	// remove all transcoded segments

	m.cancel()
	close(m.shutdown)

	m.ready = false
}

func (m *ManagerCtx) Cleanup() {
	// check what segments are really needed
	// stop transcoding processes that are not needed anymore
}

func (m *ManagerCtx) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	// ensure that transcode started
	if !m.ready {
		select {
		// waiting for transcode to be ready
		case <-m.onReadyChange:
			// check if it started succesfully
			if !m.ready {
				m.logger.Warn().Msgf("playlist load failed")
				http.Error(w, "504 playlist not available", http.StatusInternalServerError)
				return
			}
		// when transcode stops before getting ready
		case <-m.shutdown:
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

	segmentName, ok := m.segments[index]
	if !ok {
		http.Error(w, "404 index not found", http.StatusNotFound)
		return
	}

	// check if media is already transcoded
	if segmentName == "" {
		segChan, err := m.transcodeSegment(r.Context(), index)
		if err != nil {
			m.logger.Err(err).Int("index", index).Msg("unable to transcode media")
			http.Error(w, "500 unable to transcode", http.StatusInternalServerError)
			return
		}

		select {
		// waiting for new segment to be transcoded
		case segmentName = <-segChan:
		// when transcode stops before getting ready
		case <-m.shutdown:
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
	segmentPath := path.Join(m.config.TranscodeDir, segmentName)
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

func (m *ManagerCtx) OnStart(event func()) {
	m.events.onStart = event
}

func (m *ManagerCtx) OnCmdLog(event func(message string)) {
	m.events.onCmdLog = event
}

func (m *ManagerCtx) OnStop(event func(err error)) {
	m.events.onStop = event
}
