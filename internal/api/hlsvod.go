package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
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
		urlPath, err := url.PathUnescape(r.URL.Path[5:])
		if err != nil {
			logger.Error().Err(err).Msg("Failed to unescape URL path")
			http.Error(w, "Failed to unescape URL path", http.StatusBadRequest)
			return
		}

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
		vodMediaPath = path.Join(a.config.Vod.MediaDir, vodMediaPath)

		// serve master profile
		if hlsResource == "index.m3u8" {
			data, err := hlsvod.New(hlsvod.Config{
				MediaPath:      vodMediaPath,
				VideoKeyframes: a.config.Vod.VideoKeyframes,

				Cache:    a.config.Vod.Cache,
				CacheDir: a.config.Vod.CacheDir,

				FFmpegBinary:  a.config.Vod.FFmpegBinary,
				FFprobeBinary: a.config.Vod.FFprobeBinary,
			}).Preload(r.Context())

			if err != nil {
				logger.Warn().Err(err).Msg("unable to preload metadata")
				http.Error(w, "500 unable to preload metadata", http.StatusInternalServerError)
				return
			}

			width, height := 0, 0
			if data.Video != nil {
				width, height = data.Video.Width, data.Video.Height
			}

			profiles := map[string]hlsvod.VideoProfile{}
			for name, profile := range a.config.Vod.VideoProfiles {
				if width != 0 && width < profile.Width &&
					height != 0 && height < profile.Height {
					continue
				}

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

		ID := fmt.Sprintf("%s/%s", profileID, vodMediaPath)
		manager, ok := hlsVodManagers[ID]

		logger.Info().
			Str("path", urlPath).
			Str("hlsResource", hlsResource).
			Str("vodMediaPath", vodMediaPath).
			Msg("new hls vod request")

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
				logger.Warn().Err(err).Msg("could not create temp dir")
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
				VideoKeyframes: a.config.Vod.VideoKeyframes,
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
				logger.Warn().Err(err).Msg("hls vod manager could not be started")
				http.Error(w, "500 hls vod manager could not be started", http.StatusInternalServerError)
				return
			}
		}

		// server playlist or segment
		if hlsResource == profileID+".m3u8" {
			manager.ServePlaylist(w, r)
		} else {
			manager.ServeMedia(w, r)
		}
	})
}
