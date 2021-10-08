package hlsvod

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// how long can it take for transcode to be ready
const readyTimeout = 24 * time.Second

type ManagerCtx struct {
	logger zerolog.Logger
	mu     sync.Mutex
	config Config

	ready         bool
	onReadyChange chan struct{}

	events struct {
		onStart  func()
		onCmdLog func(message string)
		onStop   func(err error)
	}

	probeData       *ProbeMediaData
	segmentTimes    []float64
	segmentDuration float64

	shutdown chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(config Config) *ManagerCtx {
	ctx, cancel := context.WithCancel(context.Background())
	return &ManagerCtx{
		logger: log.With().Str("module", "hlsvod").Str("submodule", "manager").Logger(),
		config: config,

		ctx:    ctx,
		cancel: cancel,
	}
}

// TODO: Cache.
func (m *ManagerCtx) loadData() (err error) {
	// start ffprobe to get metadata about current media
	m.probeData, err = ProbeMedia(m.ctx, m.config.FFprobeBinary, m.config.MediaPath)
	if err != nil {
		return fmt.Errorf("unable probe media for metadata: %v", err)
	}

	// if media has video, use keyframes as reference for segments
	var keyframes []float64
	if m.probeData.Video != nil && m.segmentTimes == nil {
		// start ffprobe to get keyframes from video
		videoData, err := ProbeVideo(m.ctx, m.config.FFprobeBinary, m.config.MediaPath)
		if err != nil {
			return fmt.Errorf("unable probe video for keyframes: %v", err)
		}
		keyframes = videoData.PktPtsTime
	}

	// TODO: Generate segment times from keyframes.
	m.segmentTimes = keyframes
	return nil
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

	// Load data asynchronously
	go func() {
		if err := m.loadData(); err != nil {
			log.Printf("%v\n", err)
			return
		}

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

func (m *ManagerCtx) getPlaylist() string {
	// playlist prefix
	playlist := []string{
		"#EXTM3U",
		"#EXT-X-VERSION:4",
		"#EXT-X-PLAYLIST-TYPE:VOD",
		"#EXT-X-MEDIA-SEQUENCE:0",
		fmt.Sprintf("#EXT-X-TARGETDURATION:%.2f", m.segmentDuration),
	}

	// playlist segments
	for i := 1; i < len(m.segmentTimes); i++ {
		playlist = append(playlist,
			fmt.Sprintf("#EXTINF:%.3f, no desc", m.segmentTimes[i]-m.segmentTimes[i-1]),
			fmt.Sprintf("%s-%05d.ts", m.config.SegmentPrefix, i),
		)
	}

	// playlist suffix
	playlist = append(playlist,
		"#EXT-X-ENDLIST",
	)

	// join with newlines
	return strings.Join(playlist, "\n")
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

	playlist := m.getPlaylist()
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write([]byte(playlist))
}

func (m *ManagerCtx) ServeMedia(w http.ResponseWriter, r *http.Request) {
	// check if media is already transcoded
	//	- if not, check if probe data exists
	//	-	- if not, check if probe is not running
	//	-	-	- if not, start it
	//	-	- wait for it to finish
	//	- start transcoding from this segment
	//	- wait for this segment to finish
	// return existing segment
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
