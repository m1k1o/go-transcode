package hlsproxy

import (
	"time"

	"github.com/rs/zerolog/log"
)

func (m *ManagerCtx) getFromCache(key string) ([]byte, bool) {
	m.cacheMu.RLock()
	entry, ok := m.cache[key]
	m.cacheMu.RUnlock()

	// on cache miss
	if !ok {
		log.Info().Str("key", key).Msg("cache miss")
		return nil, false
	}

	// if cache has expired
	if time.Now().After(entry.Expires) {
		m.removeFromCache(key)
		return nil, false
	}

	// cache hit
	log.Info().Str("key", key).Msg("cache hit")
	return entry.Data, true
}

func (m *ManagerCtx) saveToCache(key string, data []byte, duration time.Duration) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	log.Info().Str("key", key).Msg("cache add")
	m.cache[key] = Cache{
		Data:    data,
		Expires: time.Now().Add(duration),
	}
}

func (m *ManagerCtx) removeFromCache(key string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	delete(m.cache, key)
	log.Info().Str("key", key).Msg("cache remove")
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
