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

	// start periodic cleanup if not running
	m.cleanupStart()

	return cache
}

func (m *ManagerCtx) clearCache() {
	cacheSize := 0

	m.cacheMu.Lock()
	for key, entry := range m.cache {
		// remove expired entries
		if time.Now().After(entry.Expires) {
			delete(m.cache, key)
			m.logger.Debug().Str("key", key).Msg("cache cleanup remove expired")
		} else {
			cacheSize++
		}
	}
	m.cacheMu.Unlock()

	if cacheSize == 0 {
		m.cleanupStop()
	}
}

func (m *ManagerCtx) cleanupStart() {
	m.cleanupMu.Lock()
	defer m.cleanupMu.Unlock()

	// if already running
	if m.cleanup {
		return
	}

	m.shutdown = make(chan struct{})
	m.cleanup = true

	go func() {
		m.logger.Debug().Msg("cleanup started")

		ticker := time.NewTicker(m.config.CacheCleanupPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-m.shutdown:
				return
			case <-ticker.C:
				m.logger.Debug().Msg("performing cleanup")
				m.clearCache()
			}
		}
	}()
}

func (m *ManagerCtx) cleanupStop() {
	m.cleanupMu.Lock()
	defer m.cleanupMu.Unlock()

	// if not running
	if !m.cleanup {
		return
	}

	m.cleanup = false
	close(m.shutdown)

	m.logger.Debug().Msg("cleanup stopped")
}
