package hls

import (
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/go-transcode/internal/utils"
	"github.com/m1k1o/go-transcode/internal/utils/cmdgroup"
)

// how often should be cleanup called
const cleanupPeriod = 4 * time.Second

// timeout for first playlist, when it waits for new data
const playlistTimeout = 60 * time.Second

// minimum segments available to consider stream as active
const hlsMinimumSegments = 2

// how long must be active stream idle to be considered as dead
const activeIdleTimeout = 12 * time.Second

// how long must be iactive stream idle to be considered as dead
const inactiveIdleTimeout = 24 * time.Second

type ManagerCtx struct {
	logger     zerolog.Logger
	mu         sync.Mutex
	cmdFactory func() *exec.Cmd
	active     bool
	events     struct {
		onStart  func()
		onCmdLog func(message string)
		onStop   func(err error)
	}

	cmd         *exec.Cmd
	tempdir     string
	lastRequest time.Time

	sequence int
	playlist string

	playlistLoad chan string
	shutdown     chan interface{}
}

func New(cmdFactory func() *exec.Cmd) *ManagerCtx {
	return &ManagerCtx{
		logger:     log.With().Str("module", "hls").Str("submodule", "manager").Logger(),
		cmdFactory: cmdFactory,

		playlistLoad: make(chan string),
		shutdown:     make(chan interface{}),
	}
}

func (m *ManagerCtx) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil {
		return errors.New("has already started")
	}

	m.logger.Debug().Msg("performing start")

	var err error
	m.tempdir, err = os.MkdirTemp("", "go-transcode-hls")
	if err != nil {
		return err
	}

	m.cmd = m.cmdFactory()
	m.cmd.Dir = m.tempdir

	if m.events.onCmdLog != nil {
		m.cmd.Stderr = utils.LogEvent(m.events.onCmdLog)
	} else {
		m.cmd.Stderr = utils.LogWriter(m.logger)
	}

	read, write := io.Pipe()
	m.cmd.Stdout = write

	// configure command to run in its own process group / job object
	cmdgroup.Configure(m.cmd)

	m.active = false
	m.lastRequest = time.Now()

	m.sequence = 0
	m.playlist = ""

	m.playlistLoad = make(chan string)
	m.shutdown = make(chan interface{})

	// read playlist on stdout
	go func() {
		buf := make([]byte, 1024)

		for {
			n, err := read.Read(buf)
			if n != 0 {
				m.playlist = string(buf[:n])
				m.sequence = m.sequence + 1

				m.logger.Info().
					Int("sequence", m.sequence).
					Str("playlist", m.playlist).
					Msg("received playlist")

				if m.sequence == hlsMinimumSegments {
					m.active = true
					m.playlistLoad <- m.playlist
					close(m.playlistLoad)
				}
			}

			// if stdout pipe has been closed
			if err != nil {
				m.logger.Err(err).Msg("cmd read failed")
				return
			}
		}
	}()

	// periodic cleanup
	go func() {
		ticker := time.NewTicker(cleanupPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-m.shutdown:
				write.Close()
				return
			case <-ticker.C:
				m.Cleanup()
			}
		}
	}()

	if m.events.onStart != nil {
		m.events.onStart()
	}

	// start program
	err = m.cmd.Start()

	// wait for program to exit
	go func() {
		err = m.cmd.Wait()
		if err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				// The program has exited with an exit code != 0

				// This works on both Unix and Windows. Although package
				// syscall is generally platform dependent, WaitStatus is
				// defined for both Unix and Windows and in both cases has
				// an ExitStatus() method with the same signature.
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					m.logger.Warn().Int("exit-status", status.ExitStatus()).Msg("the program has exited with an exit code != 0")
				}
			} else {
				m.logger.Err(err).Msg("the program has exited with an error")
			}
		} else {
			m.logger.Info().Msg("the program has successfully exited")
		}

		close(m.shutdown)

		if m.events.onStop != nil {
			m.events.onStop(err)
		}

		err := os.RemoveAll(m.tempdir)
		m.logger.Err(err).Msg("removing tempdir")

		m.mu.Lock()
		m.cmd = nil
		m.mu.Unlock()
	}()

	return err
}

// Stop terminates the running command and its children.
func (m *ManagerCtx) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		m.logger.Debug().Msg("performing stop")
		if err := cmdgroup.Kill(m.cmd); err != nil {
			m.logger.Err(err).Msg("failed to kill process group")
		}
	}
}

func (m *ManagerCtx) Cleanup() {
	m.mu.Lock()
	diff := time.Since(m.lastRequest)
	stop := m.active && diff > activeIdleTimeout || !m.active && diff > inactiveIdleTimeout
	m.mu.Unlock()

	m.logger.Debug().
		Time("last_request", m.lastRequest).
		Dur("diff", diff).
		Bool("active", m.active).
		Bool("stop", stop).
		Msg("performing cleanup")

	if stop {
		m.Stop()
	}
}

func (m *ManagerCtx) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.lastRequest = time.Now()
	m.mu.Unlock()

	playlist := m.playlist

	if m.cmd == nil {
		err := m.Start()
		if err != nil {
			m.logger.Warn().Err(err).Msg("transcode could not be started")
			http.Error(w, "500 not available", http.StatusInternalServerError)
			return
		}
	}

	if !m.active {
		select {
		case playlist = <-m.playlistLoad:
		// when command exits before providing any playlist
		case <-m.shutdown:
			m.logger.Warn().Msg("playlist load failed because of shutdown")
			http.Error(w, "500 playlist not available", http.StatusInternalServerError)
			return
		case <-time.After(playlistTimeout):
			m.logger.Warn().Msg("playlist load channel timeouted")
			http.Error(w, "504 playlist timeout", http.StatusGatewayTimeout)
			return
		}
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(playlist))
}

func (m *ManagerCtx) ServeMedia(w http.ResponseWriter, r *http.Request) {
	fileName := path.Base(r.URL.RequestURI())
	path := path.Join(m.tempdir, fileName)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		m.logger.Warn().Str("path", path).Msg("media file not found")
		http.Error(w, "404 media not found", http.StatusNotFound)
		return
	}

	m.mu.Lock()
	m.lastRequest = time.Now()
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, path)
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
