package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/go-transcode/pkg/hlslive"
)

var hlsManagers map[string]hlslive.Manager = make(map[string]hlslive.Manager)

//go:embed play.html
var playHTML string

func (a *ApiManagerCtx) HLS(r chi.Router) {
	r.Get("/{profile}/{input}/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		logger := log.With().Str("module", "hls").Logger()

		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")

		if !resourceRegex.MatchString(profile) || !resourceRegex.MatchString(input) {
			http.Error(w, "400 invalid parameters", http.StatusBadRequest)
			return
		}

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

		ID := fmt.Sprintf("%s/%s", profile, input)

		manager, ok := hlsManagers[ID]
		if !ok {
			// create new manager
			manager = hlslive.New(func() *exec.Cmd {
				// get transcode cmd
				cmd, err := a.transcodeStart(profilePath, input)
				if err != nil {
					logger.Error().Err(err).Msg("transcode could not be started")
				}

				return cmd
			}, nil)

			hlsManagers[ID] = manager
		}

		manager.ServePlaylist(w, r)
	})

	r.Get("/{profile}/{input}/{file}.ts", func(w http.ResponseWriter, r *http.Request) {
		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")
		file := chi.URLParam(r, "file")

		if !resourceRegex.MatchString(profile) || !resourceRegex.MatchString(input) || !resourceRegex.MatchString(file) {
			http.Error(w, "400 invalid parameters", http.StatusBadRequest)
			return
		}

		ID := fmt.Sprintf("%s/%s", profile, input)

		manager, ok := hlsManagers[ID]
		if !ok {
			http.Error(w, "404 transcode not found", http.StatusNotFound)
			return
		}

		manager.ServeMedia(w, r)
	})

	r.Get("/{profile}/{input}/play.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(playHTML))
	})
}
