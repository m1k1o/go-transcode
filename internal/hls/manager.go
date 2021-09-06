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

	"github.com/m1k1o/go-transcode/internal/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const cleanupPeriod = 2 * time.Second
const hlsMinimumSegments = 2
const hlsSegmentDuration = 6

type HlsManagerCtx struct {
	logger  zerolog.Logger
	mu      sync.Mutex
	cmd     *exec.Cmd
	active  bool
	started bool

	tempdir     string
	lastRequest int64

	sequence int
	playlist string

	playlistLoad chan string
	stopCleanup  chan interface{}
}

func New(cmd *exec.Cmd) *HlsManagerCtx {
	return &HlsManagerCtx{
		logger: log.With().Str("module", "hls").Str("submodule", "manager").Logger(),
		cmd:    cmd,

		playlistLoad: make(chan string),
		stopCleanup:  make(chan interface{}),
	}
}

func (m *HlsManagerCtx) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug().Msg("performing start")

	if m.started {
		return errors.New("has already started")
	}

	var err error
	m.tempdir, err = os.MkdirTemp("", "go-transcode-hls")
	if err != nil {
		return err
	}

	m.cmd.Dir = m.tempdir
	m.cmd.Stderr = utils.LogWriter(m.logger)

	read, write := io.Pipe()
	m.cmd.Stdout = write

	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	m.started = true
	m.lastRequest = time.Now().Unix()

	m.sequence = 0
	m.playlist = ""

	m.playlistLoad = make(chan string)
	m.stopCleanup = make(chan interface{})

	go func() {
		buf := make([]byte, 1024)

		for {
			n, err := read.Read(buf)
			if n != 0 {
				m.playlist = string(buf[:n])
				m.sequence = m.sequence + 1

				m.logger.Debug().
					Int("sequence", m.sequence).
					Str("playlist", m.playlist).
					Msg("received playlist")

				if m.sequence == hlsMinimumSegments {
					m.active = true
					m.playlistLoad <- m.playlist
					close(m.playlistLoad)
				}
			}

			if err != nil {
				m.logger.Err(err).Msg("cmd read failed")
				break
			}
		}

		write.Close()
	}()

	go func() {
		ticker := time.NewTicker(cleanupPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCleanup:
				return
			case <-ticker.C:
				m.Cleanup()
			}
		}
	}()

	return m.cmd.Start()
}

func (m *HlsManagerCtx) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug().Msg("performing stop")

	if !m.started {
		return
	}

	m.started = false
	m.stopCleanup <- struct{}{}

	if m.cmd.Process != nil {
		err := m.cmd.Process.Kill()
		m.logger.Err(err).Msg("killing proccess")
	}

	err := os.RemoveAll(m.tempdir)
	m.logger.Err(err).Msg("removing tempdir")
}

func (m *HlsManagerCtx) Cleanup() {
	diff := time.Now().Unix() - m.lastRequest
	stop := m.active && diff > 2*hlsSegmentDuration || !m.active && diff > 4*hlsSegmentDuration

	m.logger.Debug().
		Int64("last_request", m.lastRequest).
		Int64("diff", diff).
		Bool("active", m.active).
		Bool("stop", stop).
		Msg("performing cleanup")

	if stop {
		m.Stop()
	}
}

func (m *HlsManagerCtx) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	m.logger.Debug().Msg("playlist request")

	m.lastRequest = time.Now().Unix()
	playlist := m.playlist

	if !m.started {
		err := m.Start()
		if err != nil {
			m.logger.Warn().Err(err).Msg("transcode could not be started")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

	if !m.active {
		select {
		case playlist = <-m.playlistLoad:
		case <-time.After(20 * time.Second):
			m.logger.Warn().Msg("playlist load channel timeouted")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 not available"))
			return
		}
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(playlist))
}

func (m *HlsManagerCtx) ServeMedia(w http.ResponseWriter, r *http.Request) {
	fileName := path.Base(r.URL.RequestURI())
	path := path.Join(m.tempdir, fileName)

	m.logger.Debug().Str("path", path).Msg("media file request")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		m.logger.Warn().Str("path", path).Msg("media file not found")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 media not found"))
		return
	}

	m.lastRequest = time.Now().Unix()
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, path)
}
