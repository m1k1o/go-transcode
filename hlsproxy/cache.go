package hlsproxy

import (
	"io"
	"time"

	"github.com/m1k1o/go-transcode/internal/utils"
)

func (m *ManagerCtx) getFromCache(key string) (*utils.Cache, bool) {
	m.cacheMu.RLock()
	entry, ok := m.cache[key]
	m.cacheMu.RUnlock()

	// on cache miss
	if !ok {
		m.logger.Debug().Str("key", key).Msg("cache miss")
		return nil, false
	}

	// if cache has expired
	if time.Now().After(entry.Expires) {
		m.removeFromCache(key)
		return nil, false
	}

	// cache hit
	m.logger.Debug().Str("key", key).Msg("cache hit")
	return entry, true
}

func (m *ManagerCtx) saveToCache(key string, reader io.Reader, duration time.Duration) *utils.Cache {
	m.cacheMu.Lock()
	cache := utils.NewCache(time.Now().Add(duration))
	m.cache[key] = cache
	m.cacheMu.Unlock()

	// pipe reader to writer.
	go func() {
		defer cache.Close()

		_, err := io.Copy(cache, reader)
		if err != nil {
			m.logger.Err(err).Msg("error while copying to cache")
		}

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
}

func (m *ManagerCtx) clearCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	// remove expired entries
	for key, entry := range m.cache {
		if time.Now().After(entry.Expires) {
			delete(m.cache, key)
			m.logger.Debug().Str("key", key).Msg("cache cleanup remove expired")
		}
	}
}
