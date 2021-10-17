package hlsproxy

import (
	"io"
	"time"

	"github.com/m1k1o/go-transcode/internal/utils"
	"github.com/rs/zerolog/log"
)

func (m *ManagerCtx) getFromCache(key string) (*utils.Cache, bool) {
	m.cacheMu.RLock()
	entry, ok := m.cache[key]
	m.cacheMu.RUnlock()

	// on cache miss
	if !ok {
		log.Warn().Str("key", key).Msg("cache miss")
		return nil, false
	}

	// if cache has expired
	if time.Now().After(entry.Expires) {
		m.removeFromCache(key)
		return nil, false
	}

	// cache hit
	log.Info().Str("key", key).Msg("cache hit !!!!")
	return entry, true
}

func (m *ManagerCtx) saveToCache(key string, reader io.Reader, duration time.Duration) *utils.Cache {
	log.Info().Str("key", key).Msg("cache add ++++")

	m.cacheMu.Lock()
	cache := utils.NewCache(time.Now().Add(duration))
	m.cache[key] = cache
	m.cacheMu.Unlock()

	// pipe reader to writer.
	go func() {
		defer cache.Close()

		_, err := io.Copy(cache, reader)
		log.Err(err).Msg("copied to cache")

		// close reader, if it needs to be closed
		if closer, ok := reader.(io.ReadCloser); ok {
			closer.Close()
		}
	}()

	return cache
}

func (m *ManagerCtx) removeFromCache(key string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	delete(m.cache, key)
	log.Info().Str("key", key).Msg("cache remove ----")
}

func (m *ManagerCtx) clearCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	// remove expired entries
	for key, entry := range m.cache {
		if time.Now().After(entry.Expires) {
			delete(m.cache, key)
			log.Info().Str("key", key).Msg("cache cleanup remove expired")
		}
	}
}