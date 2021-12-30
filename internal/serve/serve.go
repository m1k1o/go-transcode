package serve

import (
	"os"
	"os/signal"
	"path"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/m1k1o/go-transcode/internal/server"
	"github.com/m1k1o/go-transcode/modules/hlslive"
	"github.com/m1k1o/go-transcode/modules/hlsproxy"
	"github.com/m1k1o/go-transcode/modules/hlsvod"
	"github.com/m1k1o/go-transcode/modules/httpstream"
	"github.com/m1k1o/go-transcode/modules/player"
	hlsVodPkg "github.com/m1k1o/go-transcode/pkg/hlsvod"
)

func NewCommand() *Main {
	return &Main{
		Config: &Config{},
	}
}

type Main struct {
	Config *Config

	logger     zerolog.Logger
	server     *server.ServerManagerCtx
	hlsLive    *hlslive.ModuleCtx
	hlsProxy   *hlsproxy.ModuleCtx
	hlsVod     *hlsvod.ModuleCtx
	httpStream *httpstream.ModuleCtx
	player     *player.ModuleCtx
}

func (main *Main) Preflight() {
	main.logger = log.With().Str("service", "main").Logger()
}

func (main *Main) start() {
	config := main.Config

	main.server = server.New(&server.Config{
		Bind:    config.Bind,
		Static:  config.Static,
		SSLCert: config.Cert,
		SSLKey:  config.Key,
		Proxy:   config.Proxy,
		PProf:   config.PProf,
	})

	/*
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			//nolint
			_, _ = w.Write([]byte("pong"))
		})
	*/

	if config.Vod.MediaDir != "" {
		videoProfiles := map[string]hlsVodPkg.VideoProfile{}
		for key, prof := range config.Vod.VideoProfiles {
			videoProfiles[key] = hlsVodPkg.VideoProfile{
				Width:   prof.Width,
				Height:  prof.Height,
				Bitrate: prof.Bitrate,
			}
		}

		main.hlsVod = hlsvod.New("/vod/", &hlsvod.Config{
			MediaBasePath:      config.Vod.MediaDir,
			TranscodeDir:       config.Vod.TranscodeDir,
			VideoProfiles:      videoProfiles,
			MasterPlaylistName: "index.m3u8",

			Config: hlsVodPkg.Config{
				VideoKeyframes: config.Vod.VideoKeyframes,
				AudioProfile: &hlsVodPkg.AudioProfile{
					Bitrate: config.Vod.AudioProfile.Bitrate,
				},

				Cache:    config.Vod.Cache,
				CacheDir: config.Vod.CacheDir,

				FFmpegBinary:  config.Vod.FFmpegBinary,
				FFprobeBinary: config.Vod.FFprobeBinary,
			},
		})

		main.server.Handle("/vod/", main.hlsVod)
		main.logger.Info().Str("vod-dir", config.Vod.MediaDir).Msg("static file transcoding is active")
	}

	if len(config.HlsProxy) > 0 {
		main.hlsProxy = hlsproxy.New("/hlsproxy/", &hlsproxy.Config{
			Sources: config.HlsProxy,
		})
		main.server.Handle("/hlsproxy/", main.hlsProxy)
		log.Info().Interface("hls-proxy", config.HlsProxy).Msg("hls proxy is active")
	}

	main.hlsLive = hlslive.New("/", &hlslive.Config{
		Sources:      config.Streams,
		ProfilesPath: path.Join(config.Profiles, "hls"),
		PlaylistName: "index.m3u8",
		// TOOD: Profile ends with .sh
	})
	main.server.Handle("/", main.hlsLive)
	main.logger.Info().Msg("hlsLive registered")

	// TODO: Match correct URLs.
	main.httpStream = httpstream.New("/", &httpstream.Config{
		Sources:      config.Streams,
		ProfilesPath: path.Join(config.Profiles, "http"),
		UseBufCopy:   false,
	})
	main.server.Handle("/", main.httpStream)
	main.logger.Info().Msg("httpStream registered")

	// TODO: Match correct URLs.
	main.player = player.New("/player/", &player.Config{})
	main.server.Handle("/player/", main.player)
	main.logger.Info().Msg("player registered")

	main.server.Start()
	main.logger.Info().Msgf("serving streams from basedir %s: %s", config.BaseDir, config.Streams)
}

func (main *Main) shutdown() {
	err := main.server.Shutdown()
	main.logger.Err(err).Msg("http manager shutdown")

	if main.hlsVod != nil {
		main.hlsVod.Shutdown()
		main.logger.Info().Msg("hlsVod shutdown")
	}

	if main.hlsProxy != nil {
		main.hlsProxy.Shutdown()
		main.logger.Info().Msg("hlsProxy shutdown")
	}

	if main.hlsLive != nil {
		main.hlsLive.Shutdown()
		main.logger.Info().Msg("hlsLive shutdown")
	}

	if main.httpStream != nil {
		main.httpStream.Shutdown()
		main.logger.Info().Msg("httpStream shutdown")
	}

	if main.player != nil {
		main.player.Shutdown()
		main.logger.Info().Msg("player shutdown")
	}
}

func (main *Main) Run(cmd *cobra.Command, args []string) {
	main.logger.Info().Msg("starting main server")
	main.start()
	main.logger.Info().Msg("main ready")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	sig := <-quit

	main.logger.Warn().Msgf("received %s, attempting graceful shutdown", sig)
	main.shutdown()
	main.logger.Info().Msg("shutdown complete")
}
