package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/go-transcode/internal/config"
)

var resourceRegex = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)

type ApiManagerCtx struct {
	config *config.Server
}

func New(config *config.Server) *ApiManagerCtx {
	return &ApiManagerCtx{
		config: config,
	}
}

func (manager *ApiManagerCtx) Start() {
}

func (manager *ApiManagerCtx) Shutdown() error {
	// stop all hls managers
	for _, hls := range hlsManagers {
		hls.Stop()
	}

	// stop all hls vod managers
	for _, hls := range hlsVodManagers {
		hls.Stop()
	}

	// shutdown all hls proxy managers
	for _, hls := range hlsProxyManagers {
		hls.Shutdown()
	}

	return nil
}

func (a *ApiManagerCtx) Mount(r *chi.Mux) {
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		//nolint
		_, _ = w.Write([]byte("pong"))
	})

	if a.config.Vod.MediaDir != "" {
		r.Group(a.HlsVod)
		log.Info().Str("vod-dir", a.config.Vod.MediaDir).Msg("static file transcoding is active")
	}

	if len(a.config.HlsProxy) > 0 {
		r.Group(a.HLSProxy)
		log.Info().Interface("hls-proxy", a.config.HlsProxy).Msg("hls proxy is active")
	}

	r.Group(a.HLS)
	r.Group(a.Http)
}

func (a *ApiManagerCtx) ProfilePath(folder string, profile string) (string, error) {
	// [profiles]/hls,http/[profile].sh
	// [profiles] defaults to [basedir]/profiles

	if !resourceRegex.MatchString(profile) {
		return "", fmt.Errorf("invalid profile path")
	}

	profilePath := path.Join(a.config.Profiles, folder, fmt.Sprintf("%s.sh", profile))
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return "", err
	}
	return profilePath, nil
}

// Call ProfilePath before
func (a *ApiManagerCtx) transcodeStart(profilePath string, input string) (*exec.Cmd, error) {
	url, ok := a.config.Streams[input]
	if !ok {
		return nil, fmt.Errorf("stream not found")
	}

	log.Info().Str("profilePath", profilePath).Str("url", url).Msg("command started")
	return exec.Command(profilePath, url), nil
}
