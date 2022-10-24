package hlsvod

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/m1k1o/go-transcode/pkg/hlsvod"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ModuleCtx struct {
	logger     zerolog.Logger
	pathPrefix string
	config     Config

	managers map[string]hlsvod.Manager
}

func New(pathPrefix string, config *Config) *ModuleCtx {
	module := &ModuleCtx{
		logger:     log.With().Str("module", "hlsvod").Logger(),
		pathPrefix: pathPrefix,
		config:     config.withDefaultValues(),

		managers: make(map[string]hlsvod.Manager),
	}

	return module
}

func (m *ModuleCtx) Shutdown() {
	for _, manager := range m.managers {
		manager.Stop()
	}
}

// TODO: Reload config in all managers.
func (m *ModuleCtx) ConfigReload(config *Config) {
	m.config = config.withDefaultValues()
}

// TODO: Periodically call this to remove old managers.
func (m *ModuleCtx) Cleanup() {

}

func (m *ModuleCtx) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, m.pathPrefix) {
		http.NotFound(w, r)
		return
	}

	p := r.URL.Path
	// remove path prefix
	p = strings.TrimPrefix(p, m.pathPrefix)
	// remove leading and ending /
	p = strings.Trim(p, "/")

	// get index of last slash from path
	lastSlashIndex := strings.LastIndex(p, "/")
	if lastSlashIndex == -1 {
		http.Error(w, "400 invalid parameters", http.StatusBadRequest)
		return
	}

	// everything after last slash is hls resource (playlist or segment)
	hlsResource := p[lastSlashIndex+1:]
	// everything before last slash is vod media path
	vodMediaPath := p[:lastSlashIndex]
	// use clean path
	vodMediaPath = filepath.Clean(vodMediaPath)
	vodMediaPath = path.Join(m.config.MediaBasePath, vodMediaPath)

	// serve master profile
	if hlsResource == m.config.MasterPlaylistName {
		// modify default config
		config := m.config.Config
		config.MediaPath = vodMediaPath

		data, err := hlsvod.New(&config).Preload(r.Context())
		if err != nil {
			m.logger.Warn().Err(err).Msg("unable to preload metadata")
			http.Error(w, "500 unable to preload metadata", http.StatusInternalServerError)
			return
		}

		width, height := 0, 0
		if data.Video != nil {
			width, height = data.Video.Width, data.Video.Height
		}

		profiles := map[string]hlsvod.VideoProfile{}
		for name, profile := range m.config.VideoProfiles {
			if width != 0 && width < profile.Width &&
				height != 0 && height < profile.Height {
				continue
			}

			profiles[name] = hlsvod.VideoProfile{
				Width:   profile.Width,
				Height:  profile.Height,
				Bitrate: (profile.Bitrate + m.config.AudioProfile.Bitrate) / 100 * 105000,
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
	profile, ok := m.config.VideoProfiles[profileID]
	if !ok {
		http.Error(w, "404 profile not found", http.StatusNotFound)
		return
	}

	ID := fmt.Sprintf("%s/%s", profileID, vodMediaPath)
	manager, ok := m.managers[ID]

	m.logger.Info().
		Str("path", p).
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
		transcodeDir, err := os.MkdirTemp(m.config.TranscodeDir, fmt.Sprintf("vod-%s-*", profileID))
		if err != nil {
			m.logger.Warn().Err(err).Msg("could not create temp dir")
			http.Error(w, "500 could not create temp dir", http.StatusInternalServerError)
			return
		}

		// modify default config
		config := m.config.Config
		config.MediaPath = vodMediaPath
		config.TranscodeDir = transcodeDir // with current medias subfolder
		config.SegmentNamePrefix = profileID
		config.VideoProfile = &hlsvod.VideoProfile{
			Width:   profile.Width,
			Height:  profile.Height,
			Bitrate: profile.Bitrate,
		}

		// create new manager
		manager = hlsvod.New(&config)
		if err := manager.Start(); err != nil {
			m.logger.Warn().Err(err).Msg("hls vod manager could not be started")
			http.Error(w, "500 hls vod manager could not be started", http.StatusInternalServerError)
			return
		}

		m.managers[ID] = manager
	}

	// server playlist or segment
	if hlsResource == profileID+".m3u8" {
		manager.ServePlaylist(w, r)
	} else {
		manager.ServeSegment(w, r)
	}
}
