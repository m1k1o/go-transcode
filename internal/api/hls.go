package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/go-transcode/hls"
)

var hlsManagers map[string]hls.Manager = make(map[string]hls.Manager)

//go:embed play.html
var playHTML string

func (a *ApiManagerCtx) HLS(r chi.Router) {
	r.Get("/{profile}/{input}/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		logger := log.With().
			Str("module", "m3u8").
			Logger()

		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")

		if !resourceRegex.MatchString(profile) || !resourceRegex.MatchString(input) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("400 invalid parameters"))
			return
		}

		// check if stream exists
		_, ok := a.config.Streams[input]
		if !ok {
			http.NotFound(w, r)
			return
		}

		// check if profile exists
		profilePath, err := a.ProfilePath("hls", profile)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to find profile")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%v\n", err)))
			return
		}

		ID := fmt.Sprintf("%s/%s", profile, input)

		manager, ok := hlsManagers[ID]
		if !ok {
			// create new manager
			manager = hls.New(func() *exec.Cmd {
				// get transcode cmd
				cmd, err := a.transcodeStart(profilePath, input)
				if err != nil {
					logger.Error().Err(err).Msg("transcode could not be started")
				}

				return cmd
			})

			hlsManagers[ID] = manager
		}

		manager.ServePlaylist(w, r)
	})

	r.Get("/{profile}/{input}/{file}.ts", func(w http.ResponseWriter, r *http.Request) {
		profile := chi.URLParam(r, "profile")
		input := chi.URLParam(r, "input")
		file := chi.URLParam(r, "file")

		if !resourceRegex.MatchString(profile) || !resourceRegex.MatchString(input) || !resourceRegex.MatchString(file) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("400 invalid parameters"))
			return
		}

		ID := fmt.Sprintf("%s/%s", profile, input)

		manager, ok := hlsManagers[ID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 transcode not found"))
			return
		}

		manager.ServeMedia(w, r)
	})

	r.Get("/{profile}/{input}/play.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(playHTML))
		return
	})
}
