package hlsvod

import (
	"net/http"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ManagerCtx struct {
	logger zerolog.Logger
	mu     sync.Mutex
	config Config

	active bool
	events struct {
		onStart  func()
		onCmdLog func(message string)
		onStop   func(err error)
	}

	shutdown chan struct{}
}

func New(config Config) *ManagerCtx {
	return &ManagerCtx{
		logger: log.With().Str("module", "hlsvod").Str("submodule", "manager").Logger(),
		config: config,

		shutdown: make(chan struct{}),
	}
}

func (m *ManagerCtx) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// start ffprobe to get metadata about current media
	// if video
	//	- start ffprobe to get keyframes from video

	m.active = true
	return nil
}

func (m *ManagerCtx) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// stop all transcoding processes
	// remove all transcoded segments
	close(m.shutdown)
	m.active = false
}

func (m *ManagerCtx) Cleanup() {
	// check what segments are really needed
	// stop transcoding processes that are not needed anymore
}

func (m *ManagerCtx) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	// check if probe data exists
	//	- if not, check if probe is not running
	//	-	- if not running, start it
	//	- wait for it to finish
	// return existing playlist
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
