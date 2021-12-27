package httpstream

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/m1k1o/go-transcode/internal/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var resourceRegex = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)

type ModuleCtx struct {
	logger     zerolog.Logger
	pathPrefix string
	config     Config
}

func New(pathPrefix string, config *Config) *ModuleCtx {
	module := &ModuleCtx{
		logger:     log.With().Str("module", "httpstream").Logger(),
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

	// we need exactly two parts of the url
	if len(s) != 2 {
		http.NotFound(w, r)
		return
	}

	// {source}/{profile}/{resource}
	sourceName, profileName := s[0], s[1]

	// check if parameters match regex
	if !resourceRegex.MatchString(sourceName) ||
		!resourceRegex.MatchString(profileName) {
		http.Error(w, "400 invalid parameters", http.StatusBadRequest)
		return
	}

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

	cmd := exec.CommandContext(r.Context(), profilePath, source)
	cmd.Stderr = utils.LogWriter(m.logger)

	if m.config.UseBufCopy {
		m.bufCopyCmdToHttp(cmd, w)
	} else {
		m.pipeCmdToHttp(cmd, w)
	}
}

func (m *ModuleCtx) bufCopyCmdToHttp(cmd *exec.Cmd, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "video/mp2t")

	read, write := io.Pipe()
	cmd.Stdout = write

	go utils.IOPipeToHTTP(w, read)
	m.logger.Info().Msg("starting command")

	err := cmd.Run()
	if err != nil {
		m.logger.Warn().Err(err).Msg("transcode could not be started")
		http.Error(w, "500 not available", http.StatusInternalServerError)
		return
	}

	write.Close()
	m.logger.Info().Msg("command finished")
}

func (m *ModuleCtx) pipeCmdToHttp(cmd *exec.Cmd, w http.ResponseWriter) {
	read, write := io.Pipe()
	cmd.Stdout = write

	err := cmd.Start()
	if err != nil {
		m.logger.Warn().Err(err).Msg("transcode could not be started")
		http.Error(w, "500 not available", http.StatusInternalServerError)
		return
	}

	m.logger.Info().Msg("command started")

	go func() {
		err := cmd.Wait()
		m.logger.Err(err).Msg("command finished")
		read.Close()
		write.Close()
	}()

	w.Header().Set("Content-Type", "video/mp2t")
	_, _ = io.Copy(w, read)
}
