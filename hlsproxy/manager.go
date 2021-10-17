package hlsproxy

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ManagerCtx struct {
	logger  zerolog.Logger
	mu      sync.Mutex
	baseUrl string
	prefix  string
}

func New(baseUrl string, prefix string) *ManagerCtx {
	// ensure it ends with slash
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	baseUrl += "/"

	return &ManagerCtx{
		logger:  log.With().Str("module", "hlsproxy").Str("submodule", "manager").Logger(),
		baseUrl: baseUrl,
		prefix:  prefix,
	}
}

func (m *ManagerCtx) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO.

	return nil
}

func (m *ManagerCtx) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO.
}

func (m *ManagerCtx) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	url := m.baseUrl + strings.TrimPrefix(r.URL.String(), m.prefix)

	resp, err := http.Get(url)
	if err != nil {
		log.Err(err).Msg("unable to get HTTP")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("unadle to read response body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var re = regexp.MustCompile(`(?m:^(https?\:\/\/[^\/]+)?\/)`)
	text := re.ReplaceAllString(string(buf), m.prefix)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(resp.StatusCode)

	// TODO: Cache.

	w.Write([]byte(text))
}

func (m *ManagerCtx) ServeMedia(w http.ResponseWriter, r *http.Request) {
	url := m.baseUrl + strings.TrimPrefix(r.URL.String(), m.prefix)

	resp, err := http.Get(url)
	if err != nil {
		log.Err(err).Msg("unable to get HTTP")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "video/MP2T")
	w.WriteHeader(resp.StatusCode)

	// TODO: Cache.

	io.Copy(w, resp.Body)
}
