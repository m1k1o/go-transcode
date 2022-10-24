package hlslive

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/m1k1o/go-transcode/pkg/hlslive"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var resourceRegex = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)

type ModuleCtx struct {
	logger     zerolog.Logger
	pathPrefix string
	config     Config

	managers map[string]hlslive.Manager
}

func New(pathPrefix string, config *Config) *ModuleCtx {
	module := &ModuleCtx{
		logger:     log.With().Str("module", "hlslive").Logger(),
		pathPrefix: pathPrefix,
		config:     config.withDefaultValues(),

		managers: make(map[string]hlslive.Manager),
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
	// split path to parts
	s := strings.Split(p, "/")

	// we need exactly three parts of the url
	if len(s) != 3 {
		http.NotFound(w, r)
		return
	}

	// {source}/{profile}/{resource}
	sourceName, profileName, resource := s[0], s[1], s[2]

	// check if parameters match regex
	if !resourceRegex.MatchString(sourceName) ||
		!resourceRegex.MatchString(profileName) {
		http.Error(w, "400 invalid parameters", http.StatusBadRequest)
		return
	}

	ID := fmt.Sprintf("%s/%s", sourceName, profileName)
	manager, ok := m.managers[ID]
	if !ok {
		// find relevant source
		source, ok := m.config.Sources[sourceName]
		if !ok {
			http.Error(w, "404 source not found", http.StatusNotFound)
			return
		}

		// check if exists profile path
		profilePath := path.Join(m.config.ProfilesPath, profileName)
		if _, err := os.Stat(profilePath); err != nil {
			http.Error(w, "404 profile not found", http.StatusNotFound)
			return
		}

		// create new manager
		manager = hlslive.New(func() *exec.Cmd {
			return exec.Command(profilePath, source)
		}, &m.config.Config)

		m.managers[ID] = manager
	}

	if resource == m.config.PlaylistName {
		manager.ServePlaylist(w, r)
	} else {
		manager.ServeSegment(w, r)
	}
}
