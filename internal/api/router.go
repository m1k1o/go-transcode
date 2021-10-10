package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"

	"github.com/go-chi/chi"
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

	return nil
}

func (a *ApiManagerCtx) Mount(r *chi.Mux) {
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		//nolint
		_, _ = w.Write([]byte("pong"))
	})

	if a.config.VodDir != "" {
		r.Group(a.HlsVod)
		log.Info().Str("vod-dir", a.config.VodDir).Msg("static file transcoding is active")
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

	log.Info().Str("profilePath", profilePath).Str("url", url).Msg("command startred")
	return exec.Command(profilePath, url), nil
}
