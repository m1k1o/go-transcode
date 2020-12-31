package api

import (
    "io"
    "os"
	"fmt"
	"time"
    "regexp"
	"os/exec"
	"syscall"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"
)

const (
	HLS_SEG_DURATION = 6
	HLS_MIN_SEG = 2
)

type HlsTranscode struct {
	cwd          string
	last_request int64
	cmd          *exec.Cmd

	sequence     int
	playlist     string
	active       bool
	started      chan string
}

var hlsTranscodes map[string]*HlsTranscode = make(map[string]*HlsTranscode)
var shutdown = make(chan struct{})

func init() {
	const pingPeriod = 2 * time.Second
	ticker := time.NewTicker(pingPeriod)

	go func(){
		logger := log.With().
			Str("module", "hls cleanup").
			Logger()

		defer ticker.Stop()

		for {
			select {
			case <-shutdown:
				return
			case <-ticker.C:
				logger.Debug().Msg("cleanup startred")

				now := time.Now().Unix()
				for id, profile := range hlsTranscodes {
					diff := now - profile.last_request

					logger.Debug().
						Str("id", id).
						Str("cwd", profile.cwd).
						Int64("last_request", profile.last_request).
						Int64("diff", diff).
						Msg("active profile")
					
					if profile.active && diff > 2 * HLS_SEG_DURATION || !profile.active && diff > 4 * HLS_SEG_DURATION  {
						logger.Info().
							Str("id", id).
							Msg("killing process")

						if profile.cmd.Process != nil {
							profile.cmd.Process.Kill()
						}

						if err := os.RemoveAll(profile.cwd); err != nil {
							logger.Warn().Err(err).Msg("error while directory cleanup process")
						}

						delete(hlsTranscodes, id)
					}
				}
			}
		}
	}()
}

func (a *ApiManagerCtx) HLS(r chi.Router) {
	r.Get("/{profile}/{input}/index.m3u8", func (w http.ResponseWriter, r *http.Request) {
		logger := log.With().
			Str("module", "m3u8").
			Logger()

		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")

		re := regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
		if !re.MatchString(profile) || !re.MatchString(input) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("400 invalid parameters"))
			return
		}

		ID := profile + "/" + input

		var hls *HlsTranscode
		hls, ok := hlsTranscodes[ID]
		if !ok {
			var err error

			// if transcode is not running, start
			hls, err = startHlsTranscode(ID, profile, input)
			if err != nil {
				logger.Warn().Err(err).Msg("transcode could not be started")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("%v", err)))
				return
			}
		}

		hls.last_request = time.Now().Unix()
		playlist := hls.playlist

		// if not active, wait until stream is active
		if !hls.active {
			select {
			case playlist = <- hls.started:
			case <-time.After(20 * time.Second):
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 not available"))
				return
			}
		}

		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte(playlist))
		return
	})

	r.Get("/{profile}/{input}/{file}.ts", func (w http.ResponseWriter, r *http.Request) {
		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")
		file := chi.URLParam(r, "file")

		re := regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
		if !re.MatchString(profile) || !re.MatchString(input) || !re.MatchString(file) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("400 invalid parameters"))
			return
		}

		ID := profile + "/" + input

		hls, ok := hlsTranscodes[ID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 transcode not found"))
			return
		}

		path := hls.cwd + "/" + file + ".ts"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 media not found"))
			return
		}

		hls.last_request = time.Now().Unix()
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, path)
	})

	r.Get("/{profile}/{input}/play.html", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.ServeFile(w, r, "/app/play.html")
	})
}

func startHlsTranscode(id string, profile string, input string) (*HlsTranscode, error) {
	logger := log.With().
		Str("module", "hls").
		Logger()

	cwd := "/tmp/transcodes/" + profile + "/" + input
	err := os.MkdirAll(cwd, 0755)
	if err != nil {
		return nil, err
	}

	cmd, err := transcodeStart("profiles/hls", profile, input)
	if err != nil {
		return nil, err
	}

	cmd.Dir = cwd
	cmd.Stderr = NewLogWriter(logger)
	
	read, write := io.Pipe()
	cmd.Stdout = write

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	hls := &HlsTranscode{
		cwd:          cwd,
		last_request: time.Now().Unix(),
		cmd:          cmd,
		sequence:     0,
		playlist:     "",
		started:      make(chan string),
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := read.Read(buf)
			if n != 0 {
				hls.playlist = string(buf[:n])
				hls.sequence = hls.sequence + 1

				logger.Debug().
					Int("sequence", hls.sequence).
					Str("str", hls.playlist).
					Msg("received playlist")

				if hls.sequence == HLS_MIN_SEG {
					hls.active = true
					hls.started <- hls.playlist
					close(hls.started)
				}
			}

			if err != nil {
				break
			}
		}

		logger.Info().Msg("Goroutine finished")
		write.Close()
	}()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	hlsTranscodes[id] = hls
	return hls, nil
}
