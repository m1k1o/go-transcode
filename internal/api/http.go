package api

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/go-transcode/internal/utils"
)

func (a *ApiManagerCtx) Http(r chi.Router) {
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")
		logger := log.With().
			Str("path", r.URL.Path).
			Str("module", "ffmpeg").
			Logger()

		logger.Info().Msg("command startred")
		// WTF is this for?
		cmd := exec.Command(path.Join(a.Conf.BaseDir, "data/http-test.sh"))

		read, write := io.Pipe()
		cmd.Stdout = write
		cmd.Stderr = utils.LogWriter(logger)

		defer func() {
			logger.Info().Msg("command stopped")

			read.Close()
			write.Close()
		}()

		go cmd.Run()
		io.Copy(w, read)
	})

	r.Get("/{profile}/{input}", func(w http.ResponseWriter, r *http.Request) {
		logger := log.With().
			Str("path", r.URL.Path).
			Str("module", "ffmpeg").
			Logger()

		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")

		_, stream_exists := a.Conf.Streams[input]
		if !stream_exists {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 not found"))
			return
		}

		cmd, err := a.transcodeStart("http", profile, input)
		if err != nil {
			logger.Warn().Err(err).Msg("transcode could not be started")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%v", err)))
			return
		}

		logger.Info().Msg("command started")
		w.Header().Set("Content-Type", "video/mp2t")

		read, write := io.Pipe()
		cmd.Stdout = write
		cmd.Stderr = utils.LogWriter(logger)

		defer func() {
			logger.Info().Msg("command stopped")

			read.Close()
			write.Close()
		}()

		go cmd.Run()
		io.Copy(w, read)
	})

	r.Get("/{profile}/{input}/buf", func(w http.ResponseWriter, r *http.Request) {
		logger := log.With().
			Str("path", r.URL.Path).
			Str("module", "ffmpeg").
			Logger()

		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")

		// TODO: what is this? why no http/hls?
		cmd, err := a.transcodeStart("profiles", profile, input)
		if err != nil {
			logger.Warn().Err(err).Msg("transcode could not be started")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%v", err)))
			return
		}

		logger.Info().Msg("command started")
		w.Header().Set("Content-Type", "video/mp2t")

		read, write := io.Pipe()
		cmd.Stdout = write
		cmd.Stderr = utils.LogWriter(logger)

		go utils.IOPipeToHTTP(w, read)
		cmd.Run()
		write.Close()
		logger.Info().Msg("command stopped")
	})
}
