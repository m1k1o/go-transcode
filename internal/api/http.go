package api

import (
	"io"
	"net/http"
	"os/exec"

	"github.com/go-chi/chi/v5"
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

		// dummy input for testing purposes
		file := a.config.AbsPath("profiles", "http-test.sh")
		cmd := exec.Command(file)
		logger.Info().Msg("command started")

		read, write := io.Pipe()
		cmd.Stdout = write
		cmd.Stderr = utils.LogWriter(logger)

		defer func() {
			logger.Info().Msg("command stopped")

			read.Close()
			write.Close()
		}()

		go func() {
			_ = cmd.Run()
		}()
		_, _ = io.Copy(w, read)
	})

	r.Get("/{profile}/{input}", func(w http.ResponseWriter, r *http.Request) {
		logger := log.With().
			Str("path", r.URL.Path).
			Str("module", "ffmpeg").
			Logger()

		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")

		// check if stream exists
		_, ok := a.config.Streams[input]
		if !ok {
			http.Error(w, "404 stream not found", http.StatusNotFound)
			return
		}

		// check if profile exists
		profilePath, err := a.ProfilePath("hls", profile)
		if err != nil {
			logger.Warn().Err(err).Msg("profile path could not be found")
			http.Error(w, "404 profile not found", http.StatusNotFound)
			return
		}

		cmd, err := a.transcodeStart(profilePath, input)
		if err != nil {
			logger.Warn().Err(err).Msg("transcode could not be started")
			http.Error(w, "500 not available", http.StatusInternalServerError)
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

		go func() {
			_ = cmd.Run()
		}()
		_, _ = io.Copy(w, read)
	})

	// buffered http streaming (alternative to prervious type)
	r.Get("/{profile}/{input}/buf", func(w http.ResponseWriter, r *http.Request) {
		logger := log.With().
			Str("path", r.URL.Path).
			Str("module", "ffmpeg").
			Logger()

		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")

		// check if stream exists
		_, ok := a.config.Streams[input]
		if !ok {
			http.Error(w, "404 stream not found", http.StatusNotFound)
			return
		}

		// check if profile exists
		profilePath, err := a.ProfilePath("hls", profile)
		if err != nil {
			logger.Warn().Err(err).Msg("profile path could not be found")
			http.Error(w, "404 profile not found", http.StatusNotFound)
			return
		}

		cmd, err := a.transcodeStart(profilePath, input)
		if err != nil {
			logger.Warn().Err(err).Msg("transcode could not be started")
			http.Error(w, "500 not available", http.StatusInternalServerError)
			return
		}

		logger.Info().Msg("command started")
		w.Header().Set("Content-Type", "video/mp2t")

		read, write := io.Pipe()
		cmd.Stdout = write
		cmd.Stderr = utils.LogWriter(logger)

		go utils.IOPipeToHTTP(w, read)
		_ = cmd.Run()
		write.Close()
		logger.Info().Msg("command stopped")
	})
}
