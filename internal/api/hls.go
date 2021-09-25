package api

import (
	"fmt"
	"net/http"
	"os/exec"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/go-transcode/hls"
)

var hlsManagers map[string]hls.Manager = make(map[string]hls.Manager)

func (a *ApiManagerCtx) HLS(r chi.Router) {
	r.Get("/{profile}/{input}/index.m3u8", func(w http.ResponseWriter, r *http.Request) {
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

		_, stream_exists := a.Conf.Streams[input]
		if !stream_exists {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 not found"))
			return
		}

		ID := fmt.Sprintf("%s/%s", profile, input)

		manager, ok := hlsManagers[ID]
		if !ok {
			// create new manager
			manager = hls.New(func() *exec.Cmd {
				// get transcode cmd
				cmd, err := a.transcodeStart("hls", profile, input)
				if err != nil {
					logger.Error().Err(err).Msg("transcode could not be started")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(fmt.Sprintf("%v\n", err)))
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

		re := regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
		if !re.MatchString(profile) || !re.MatchString(input) || !re.MatchString(file) {
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
		http.ServeFile(w, r, fmt.Sprintf("%s/%s", a.Conf.BaseDir, "data/play.html"))
	})
}
