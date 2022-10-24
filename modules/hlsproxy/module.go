package hlsproxy

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/m1k1o/go-transcode/pkg/hlsproxy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var resourceRegex = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)

type ModuleCtx struct {
	logger     zerolog.Logger
	pathPrefix string
	config     Config

	managers map[string]hlsproxy.Manager
}

func New(pathPrefix string, config *Config) *ModuleCtx {
	module := &ModuleCtx{
		logger:     log.With().Str("module", "hlsproxy").Logger(),
		pathPrefix: pathPrefix,
		config:     config.withDefaultValues(),

		managers: make(map[string]hlsproxy.Manager),
	}

	return module
}

func (m *ModuleCtx) Shutdown() {
	for _, manager := range m.managers {
		manager.Shutdown()
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
	// remove leading /
	p = strings.TrimLeft(p, "/")
	// split path to parts
	s := strings.Split(p, "/")

	// we need at least first part of the url
	if len(s) == 0 {
		http.NotFound(w, r)
		return
	}

	sourceName := s[0]

	// check if parameters match regex
	if !resourceRegex.MatchString(sourceName) {
		http.Error(w, "400 invalid parameters", http.StatusBadRequest)
		return
	}

	manager, ok := m.managers[sourceName]
	if !ok {
		// find relevant source
		source, ok := m.config.Sources[sourceName]
		if !ok {
			http.Error(w, "404 source not found", http.StatusNotFound)
			return
		}

		config := m.config.Config
		config.PlaylistBaseUrl = source
		config.PlaylistPathPrefix = strings.TrimRight(m.pathPrefix, "/") + sourceName

		// create new manager
		manager = hlsproxy.New(&config)
		m.managers[sourceName] = manager
	}

	if strings.HasSuffix(r.URL.Path, ".m3u8") {
		manager.ServePlaylist(w, r)
	} else {
		manager.ServeSegment(w, r)
	}
}
