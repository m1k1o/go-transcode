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

type ManagerCtx struct {
	logger zerolog.Logger
	config Config

	ready     bool
	readyMu   sync.RWMutex
	readyChan chan struct{}

	metadata    *ProbeMediaData
	playlist    string    // m3u8 playlist string
	breakpoints []float64 // list of breakpoints for segments

	segments   map[int]string // map of segments and their filename
	segmentsMu sync.RWMutex

	segmentQueue   map[int]chan struct{} // map of segments and signaling channel for finished transcoding
	segmentQueueMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

func New(config *Config) *ManagerCtx {
	ctx, cancel := context.WithCancel(context.Background())

	return &ManagerCtx{
		logger: log.With().Str("module", "hlsvod").Str("submodule", "manager").Logger(),
		config: config.withDefaultValues(),
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

func (m *ManagerCtx) httpEnsureReady(w http.ResponseWriter) bool {
	// ensure that transcode started
	if !m.isReady() {
		select {
		// waiting for transcode to be ready
		case <-m.waitForReady():
			// check if it started succesfully
			if !m.isReady() {
				m.logger.Warn().Msgf("manager is not ready")
				http.Error(w, "500 manager not available", http.StatusInternalServerError)
				return false
			}
		// when transcode stops before getting ready
		case <-m.ctx.Done():
			m.logger.Warn().Msg("manager load failed because of shutdown")
			http.Error(w, "500 manager not available", http.StatusInternalServerError)
			return false
		case <-time.After(m.config.ReadyTimeout):
			m.logger.Warn().Msg("manager load timeouted")
			http.Error(w, "504 manager timeout", http.StatusGatewayTimeout)
			return false
		}
	}

	return true
}

//
// metadata
//

// fetch metadata using ffprobe
func (m *ManagerCtx) fetchMetadata(ctx context.Context) (err error) {
	start := time.Now()
	m.logger.Info().Msg("fetching metadata")

	// start ffprobe to get metadata about current media
	m.metadata, err = ProbeMedia(ctx, m.config.FFprobeBinary, m.config.MediaPath)
	if err != nil {
		return fmt.Errorf("unable probe media for metadata: %v", err)
	}

	// if media has video, use keyframes as reference for segments if allowed so
	if m.metadata.Video != nil && m.metadata.Video.PktPtsTime == nil && m.config.VideoKeyframes {
		// start ffprobe to get keyframes from video
		videoData, err := ProbeVideo(ctx, m.config.FFprobeBinary, m.config.MediaPath)
		if err != nil {
			return fmt.Errorf("unable probe video for keyframes: %v", err)
		}
		m.metadata.Video.PktPtsTime = videoData.PktPtsTime
	}

	elapsed := time.Since(start)
	m.logger.Info().Interface("duration", elapsed).Msg("fetched metadata")
	return
}

// load metadata from cache or fetch them and cache
func (m *ManagerCtx) loadMetadata(ctx context.Context) error {
	// bypass cache if not enabled
	if !m.config.Cache {
		return m.fetchMetadata(ctx)
	}

	// try to get cached data
	data, err := m.getCacheData()
	if err == nil {
		// unmarshall cache data
		err := json.Unmarshal(data, &m.metadata)
		if err == nil {
			return nil
		}

		m.logger.Err(err).Msg("cache unmarhalling returned error, replacing")
	} else if !errors.Is(err, os.ErrNotExist) {
		m.logger.Err(err).Msg("cache hit returned error, replacing")
	}

	// fetch fresh metadata from a file
	if err := m.fetchMetadata(ctx); err != nil {
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
		fmt.Sprintf("#EXT-X-TARGETDURATION:%.2f", m.config.SegmentLength+m.config.SegmentOffset),
	}

	// playlist segments
	for i := 1; i < len(m.breakpoints); i++ {
		playlist = append(playlist,
			fmt.Sprintf("#EXTINF:%.3f, no desc", m.breakpoints[i]-m.breakpoints[i-1]),
			m.getSegmentName(i-1),
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
	m.breakpoints = convertToSegments(keyframes, m.metadata.Duration, m.config.SegmentLength, m.config.SegmentOffset)

	// generate playlist
	m.playlist = m.getPlaylist()

	// prepare transcode matrix from breakpoints
	m.segments = map[int]string{}
	for i := 0; i < len(m.breakpoints); i++ {
		m.segments[i] = ""
	}

	// prepare segment queue map
	m.segmentQueue = map[int]chan struct{}{}

	m.logger.Info().
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

func (m *ManagerCtx) getSegment(index int) (segmentPath string, ok bool) {
	var segmentName string

	m.segmentsMu.RLock()
	segmentName, ok = m.segments[index]
	m.segmentsMu.RUnlock()

	if !ok {
		return
	}

	if segmentName != "" {
		segmentPath = path.Join(m.config.TranscodeDir, segmentName)
	}

	return
}

func (m *ManagerCtx) isSegmentTranscoded(index int) bool {
	m.segmentsMu.RLock()
	segmentName, ok := m.segments[index]
	m.segmentsMu.RUnlock()

	return ok && segmentName != ""
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
			m.logger.Err(err).Str("path", segmentPath).Msg("error while removing file")
		}
	}
}

//
// segment queue
//

func (m *ManagerCtx) enqueueSegments(offset, limit int) {
	m.segmentQueueMu.Lock()
	defer m.segmentQueueMu.Unlock()

	// create new segment signaling channels queue
	for i := offset; i < offset+limit; i++ {
		m.segmentQueue[i] = make(chan struct{}, 1)
	}
}

func (m *ManagerCtx) dequeueSegment(index int) {
	m.segmentQueueMu.Lock()
	defer m.segmentQueueMu.Unlock()

	if res, ok := m.segmentQueue[index]; ok {
		close(res)
		delete(m.segmentQueue, index)
	}
}

func (m *ManagerCtx) waitForSegment(index int) (chan struct{}, bool) {
	m.segmentQueueMu.RLock()
	defer m.segmentQueueMu.RUnlock()

	res, ok := m.segmentQueue[index]
	return res, ok
}

func (m *ManagerCtx) transcodeSegments(offset, limit int) error {
	logger := m.logger.With().Int("offset", offset).Int("limit", limit).Logger()

	segmentTimes := m.breakpoints[offset : offset+limit+1]
	logger.Info().Interface("segments-times", segmentTimes).Msg("transcoding segments")

	segments, err := TranscodeSegments(m.ctx, m.config.FFmpegBinary, TranscodeConfig{
		InputFilePath: m.config.MediaPath,
		OutputDirPath: m.config.TranscodeDir,
		SegmentPrefix: m.config.SegmentPrefix, // This does not need to match.

		VideoProfile: m.config.VideoProfile,
		AudioProfile: m.config.AudioProfile,

		SegmentOffset: offset,
		SegmentTimes:  segmentTimes,
	})

	if err != nil {
		logger.Err(err).Msg("error occured while starting to transcode segment")
		return err
	}

	// create new segment signaling channels queue
	m.enqueueSegments(offset, limit)

	index := offset
	logger.Info().Msg("transcode process started")

	go func() {
		for {
			segmentName, ok := <-segments
			if !ok {
				logger.Info().Int("index", index).Msg("transcode process finished")
				return
			}

			logger.Info().
				Int("index", index).
				Str("segment", segmentName).
				Msg("transcode process returned a segment")

			// add transcoded segment name
			m.addSegment(index, segmentName)

			// notify and drop from queue, if exists
			m.dequeueSegment(index)

			// expect new segment to come
			index++
		}
	}()

	return nil
}

func (m *ManagerCtx) transcodeFromSegment(index int) error {
	segmentsTotal := len(m.segments)
	if index+m.config.SegmentBufferMax < segmentsTotal {
		segmentsTotal = index + m.config.SegmentBufferMax
	}

	offset, limit := 0, 0
	for i := index; i < segmentsTotal; i++ {
		_, isEnqueued := m.waitForSegment(i)
		isTranscoded := m.isSegmentTranscoded(i)

		// increase offset if transcoded without limit
		if (isTranscoded || isEnqueued) && limit == 0 {
			offset++
		} else
		// increase limit if is not transcoded
		if !(isTranscoded || isEnqueued) {
			limit++
		} else
		// break otherwise
		{
			break
		}
	}

	// if offset is greater than our minimal offset, we have enough segments available
	if offset > m.config.SegmentBufferMin {
		return nil
	}

	// otherwise transcode chosen segment range
	return m.transcodeSegments(offset+index, limit)
}

func (m *ManagerCtx) Start() (err error) {
	// create new executing context
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// initialize ready state
	m.readyReset()

	// initialize transcoder asynchronously
	go func() {
		if err := m.loadMetadata(m.ctx); err != nil {
			m.logger.Err(err).Msg("unable to load metadata")
			return
		}

		// initialization based on metadata
		m.initialize()

		// set ready state as done
		m.readyDone()
	}()

	return nil
}

func (m *ManagerCtx) Stop() {
	// reset ready state
	m.readyReset()

	// cancel current context
	m.cancel()

	// remove all transcoded segments
	m.clearAllSegments()
}

func (m *ManagerCtx) Preload(ctx context.Context) (*ProbeMediaData, error) {
	if err := m.loadMetadata(ctx); err != nil {
		return nil, err
	}

	return m.metadata, nil
}

func (m *ManagerCtx) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	// ensure that manager started
	if !m.httpEnsureReady(w) {
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write([]byte(m.playlist))
}

func (m *ManagerCtx) ServeSegment(w http.ResponseWriter, r *http.Request) {
	// ensure that manager started
	if !m.httpEnsureReady(w) {
		return
	}

	// same of the requested segment is everything after last slash
	reqSegName := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

	// getting index from segment name
	index, ok := m.parseSegmentIndex(reqSegName)
	if !ok {
		http.Error(w, "400 bad segment path", http.StatusBadRequest)
		return
	}

	// check if segment exists
	segmentPath, ok := m.getSegment(index)
	if !ok {
		http.Error(w, "404 index not found", http.StatusNotFound)
		return
	}

	// try to transcode from current segment
	if err := m.transcodeFromSegment(index); err != nil {
		m.logger.Err(err).Int("index", index).Msg("unable to transcode segment")
		http.Error(w, "500 unable to transcode segment", http.StatusInternalServerError)
		return
	}

	// check if segment is transcoded
	if !m.isSegmentTranscoded(index) {
		// check if segment transcoding is already in progress
		segChan, ok := m.waitForSegment(index)
		if !ok {
			// this should never happen
			m.logger.Error().Int("index", index).Msg("segment not queued even after transcode")
			http.Error(w, "409 segment not queued even after transcode", http.StatusConflict)
			return
		}

		select {
		// waiting for new segment to be transcoded
		case <-segChan:
			// now segment should be available
			segmentPath, ok = m.getSegment(index)
			if !ok || segmentPath == "" {
				// this should never happen
				m.logger.Error().Int("index", index).Msg("segment not found even after transcoding")
				http.Error(w, "409 segment not found even after transcoding", http.StatusConflict)
				return
			}
		// when transcode stops before getting ready
		case <-m.ctx.Done():
			m.logger.Warn().Msg("segment transcode failed because of shutdown")
			http.Error(w, "500 segment not available", http.StatusInternalServerError)
			return
		case <-time.After(m.config.TranscodeTimeout):
			m.logger.Warn().Msg("segment transcode timeouted")
			http.Error(w, "504 segment transcode timeout", http.StatusGatewayTimeout)
			return
		}
	}

	// check if segment is on the disk
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		m.logger.Warn().Int("index", index).Str("path", segmentPath).Msg("segment file not found")
		http.Error(w, "404 segment not found", http.StatusNotFound)
		return
	}

	// return existing segment
	w.Header().Set("Content-Type", "video/MP2T")
	http.ServeFile(w, r, segmentPath)
}
