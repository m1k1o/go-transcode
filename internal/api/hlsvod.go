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

		// serve master profile
		if hlsResource == "index.m3u8" {
			profiles := map[string]hlsvod.VideoProfile{}
			for name, profile := range a.config.Vod.VideoProfiles {
				profiles[name] = hlsvod.VideoProfile{
					Width:   profile.Width,
					Height:  profile.Height,
					Bitrate: (profile.Bitrate + a.config.Vod.AudioProfile.Bitrate) / 100 * 105000,
				}
			}

			playlist := hlsvod.StreamsPlaylist(profiles, "%s.m3u8")
			_, _ = w.Write([]byte(playlist))
			return
		}

		// get profile name (everythinb before . or -)
		profileID := strings.FieldsFunc(hlsResource, func(r rune) bool {
			return r == '.' || r == '-'
		})[0]

		// check if exists profile and fetch
		profile, ok := a.config.Vod.VideoProfiles[profileID]
		if !ok {
			http.Error(w, "404 profile not found", http.StatusNotFound)
			return
		}

		// use clean path
		vodMediaPath = filepath.Clean(vodMediaPath)
		vodMediaPath = path.Join(a.config.Vod.MediaDir, vodMediaPath)

		ID := fmt.Sprintf("%s/%s", profileID, vodMediaPath)
		manager, ok := hlsVodManagers[ID]

		logger.Info().
			Str("path", urlPath).
			Str("hlsResource", hlsResource).
			Str("vodMediaPath", vodMediaPath).
			Msg("new hls vod request")

		// if found and is not playlist request, server media
		if ok && hlsResource != profileID+".m3u8" {
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

			// create own transcoding directory
			transcodeDir, err := os.MkdirTemp(a.config.Vod.TranscodeDir, fmt.Sprintf("vod-%s-*", profileID))
			if err != nil {
				http.Error(w, "500 could not create temp dir", http.StatusInternalServerError)
				return
			}

			// create new manager
			manager = hlsvod.New(hlsvod.Config{
				MediaPath:     vodMediaPath,
				TranscodeDir:  transcodeDir,
				SegmentPrefix: profileID,

				VideoProfile: &hlsvod.VideoProfile{
					Width:   profile.Width,
					Height:  profile.Height,
					Bitrate: profile.Bitrate,
				},
				AudioProfile: &hlsvod.AudioProfile{
					Bitrate: a.config.Vod.AudioProfile.Bitrate,
				},

				Cache:    a.config.Vod.Cache,
				CacheDir: a.config.Vod.CacheDir,

				FFmpegBinary:  a.config.Vod.FFmpegBinary,
				FFprobeBinary: a.config.Vod.FFprobeBinary,
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
