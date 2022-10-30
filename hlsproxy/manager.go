package hlsproxy

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/m1k1o/go-transcode/internal/utils"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// how often should be cache cleanup called
const cacheCleanupPeriod = 4 * time.Second

// how long should be segment kept in memory
const segmentExpiration = 60 * time.Second

// how long should be playlist kept in memory
const playlistExpiration = 1 * time.Second

type ManagerCtx struct {
	logger  zerolog.Logger
	baseUrl string
	prefix  string

	cache   map[string]*utils.Cache
	cacheMu sync.RWMutex

	cleanup   bool
	cleanupMu sync.RWMutex
	shutdown  chan struct{}
}

func New(baseUrl string, prefix string) *ManagerCtx {
	// ensure it ends with slash
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	baseUrl += "/"

	return &ManagerCtx{
		logger:  log.With().Str("module", "hlsproxy").Str("submodule", "manager").Logger(),
		baseUrl: baseUrl,
		prefix:  prefix,
		cache:   map[string]*utils.Cache{},
	}
}

func (m *ManagerCtx) Shutdown() {
	m.cleanupStop()
}

func (m *ManagerCtx) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	url := m.baseUrl + strings.TrimPrefix(r.URL.String(), m.prefix)

	cache, ok := m.getFromCache(url)
	if !ok {
		resp, err := http.Get(url)
		if err != nil {
			m.logger.Err(err).Msg("unable to get HTTP")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if resp.StatusCode < 200 && resp.StatusCode >= 300 {
			// read all response body
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			m.logger.Err(err).Int("code", resp.StatusCode).Msg("invalid HTTP response")
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		// replace all urls in playlist with relative ones
		text := PlaylistUrlWalk(resp.Body, func(u string) string {
			return RelativePath(m.baseUrl, m.prefix, u)
		})

		cache = m.saveToCache(url, strings.NewReader(text), time.Now().Add(playlistExpiration))
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(200)

	cache.ServeHTTP(w)
}

func (m *ManagerCtx) ServeMedia(w http.ResponseWriter, r *http.Request) {
	url := m.baseUrl + strings.TrimPrefix(r.URL.String(), m.prefix)

	cache, ok := m.getFromCache(url)
	if !ok {
		resp, err := http.Get(url)
		if err != nil {
			m.logger.Err(err).Msg("unable to get HTTP")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if resp.StatusCode < 200 && resp.StatusCode >= 300 {
			// read all response body
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			m.logger.Err(err).Int("code", resp.StatusCode).Msg("invalid HTTP response")
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		cache = m.saveToCache(url, resp.Body, time.Now().Add(segmentExpiration))
	}

	w.Header().Set("Content-Type", "video/MP2T")
	w.WriteHeader(200)

	cache.ServeHTTP(w)
}

// resolve path: remove ../ and ./ from path
func resolvePath(path string) string {
	parts := strings.Split(path, "/")
	resolved := []string{}

	for _, part := range parts {
		if part == ".." {
			if len(resolved) > 0 {
				resolved = resolved[:len(resolved)-1]
			}
		} else if part == "." {
			continue
		} else {
			resolved = append(resolved, part)
		}
	}

	return strings.Join(resolved, "/")
}

// simple relative path resolver
func RelativePath(baseUrl, prefix, u string) string {
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		// replace base url with prefix
		u = strings.Replace(u, baseUrl, prefix, 1)
		u = resolvePath(u)
		return u
	}

	u = resolvePath(u)

	if strings.HasPrefix(u, "/") {
		// add prefix
		return strings.TrimRight(prefix, "/") + u
	}

	// we expect this to already be relative
	return u
}

// Walks playlist and replaces all urls with callback
func PlaylistUrlWalk(reader io.Reader, replace func(string) string) string {
	var sb strings.Builder

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// remove leading and trailing spaces
		line = strings.TrimSpace(line)

		// if line is empty, ignore it
		if line == "" {
			sb.WriteRune('\n')
			continue
		}

		// if line starts with #, try to find URI="..." in it
		if strings.HasPrefix(line, "#") {
			// split string by URI="
			parts1 := strings.SplitN(line, "URI=\"", 2)

			// if we don't have 2 parts, we don't have URI="..."
			if len(parts1) != 2 {
				sb.WriteString(line)
				sb.WriteRune('\n')
				continue
			}

			// split the rest of the string by " and get the first part
			parts2 := strings.SplitN(parts1[1], "\"", 2)

			// if we don't have 2 parts, something is wrong
			if len(parts2) != 2 {
				sb.WriteString(line)
				sb.WriteRune('\n')
				continue
			}

			// repalce url
			relUrl := replace(parts2[0])
			line = parts1[0] + "URI=\"" + relUrl + "\"" + parts2[1]

			sb.WriteString(line)
			sb.WriteRune('\n')
			continue
		}

		// whole line is url
		line = replace(line)
		sb.WriteString(line)
		sb.WriteRune('\n')
	}

	// close reader, if it needs to be closed
	if closer, ok := reader.(io.ReadCloser); ok {
		closer.Close()
	}

	return sb.String()
}
