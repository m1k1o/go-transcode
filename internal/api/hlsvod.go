package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	"github.com/m1k1o/go-transcode/hlsvod"
	"github.com/rs/zerolog/log"
)

var hlsVodManagers map[string]hlsvod.Manager = make(map[string]hlsvod.Manager)

func (a *ApiManagerCtx) HlsVod(r chi.Router) {
	r.Get("/vod/*", func(w http.ResponseWriter, r *http.Request) {
		logger := log.With().Str("module", "hlsvod").Logger()
		// TODO: Multi-profile.
		profile := "default"

		// remove /vod/ from path
		urlPath := r.URL.Path[5:]

		// get index of last slash from path
		lastSlashIndex := strings.LastIndex(urlPath, "/")
		if lastSlashIndex == -1 {
			http.Error(w, "400 invalid parameters", http.StatusBadRequest)
			return
		}

		// everything after last slash is hls resource (playlist or segment)
		hlsResource := urlPath[lastSlashIndex+1:]
		// everything before last slash is vod media path
		vodMediaPath := urlPath[:lastSlashIndex]

		// use clean path
		vodMediaPath = filepath.Clean(vodMediaPath)
		vodMediaPath = path.Join(a.config.VodDir, vodMediaPath)

		ID := fmt.Sprintf("%s/%s", profile, vodMediaPath)
		manager, ok := hlsVodManagers[ID]

		logger.Info().
			Str("path", urlPath).
			Str("hlsResource", hlsResource).
			Str("vodMediaPath", vodMediaPath).
			Msg("new hls vod request")

		// if found and is not playlist request, server media
		if ok && hlsResource != profile+".m3u8" {
			manager.ServeMedia(w, r)
			return
		}

		// if manager was not found
		if !ok {
			// check if vod media path exists
			if _, err := os.Stat(vodMediaPath); os.IsNotExist(err) {
				http.Error(w, "404 vod not found", http.StatusNotFound)
				return
			}

			// create new manager
			manager = hlsvod.New(hlsvod.Config{
				MediaPath: vodMediaPath,
				// TODO: Move to Config.
				TranscodeDir:  "bin/out",
				SegmentPrefix: profile,

				// TODO: Move to Config and make dependent on profile.
				VideoProfile: &hlsvod.VideoProfile{
					Width:   1280,
					Height:  720,
					Bitrate: 4200,
				},
				AudioProfile: &hlsvod.AudioProfile{
					Bitrate: 128,
				},

				// TODO: Move to Config.
				Cache: true,

				// TODO: Move to Config.
				FFmpegBinary:  "ffmpeg",
				FFprobeBinary: "ffprobe",
			})

			hlsVodManagers[ID] = manager

			if err := manager.Start(); err != nil {
				http.Error(w, "500 hls vod manager could not be started", http.StatusInternalServerError)
				return
			}
		}

		manager.ServePlaylist(w, r)
	})
}
