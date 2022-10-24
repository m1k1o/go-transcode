package player

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed player.html
var playHTML string

type ModuleCtx struct {
	logger     zerolog.Logger
	pathPrefix string
	config     Config
}

func New(pathPrefix string, config *Config) *ModuleCtx {
	module := &ModuleCtx{
		logger:     log.With().Str("module", "player").Logger(),
		pathPrefix: pathPrefix,
		config:     config.withDefaultValues(),
	}

	return module
}

func (m *ModuleCtx) Shutdown() {

}

// TODO: Reload config in all managers.
func (m *ModuleCtx) ConfigReload(config *Config) {
	m.config = config.withDefaultValues()
}

// TODO: Periodically call this to remove old managers.
func (m *ModuleCtx) Cleanup() {

}

func (m *ModuleCtx) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	html := strings.Replace(playHTML, "index.m3u8", m.config.Source, 1)
	_, _ = w.Write([]byte(html))
}
