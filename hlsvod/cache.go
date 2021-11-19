package hlsvod

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path"
)

const cacheFileSuffix = ".go-transcode-cache"

func (m *ManagerCtx) getCacheData() ([]byte, error) {
	// check for local cache
	localCachePath := m.config.MediaPath + cacheFileSuffix
	if _, err := os.Stat(localCachePath); err == nil {
		m.logger.Info().Str("path", localCachePath).Msg("media local cache hit")
		return os.ReadFile(localCachePath)
	}

	// check for global cache
	h := sha1.New()
	h.Write([]byte(m.config.MediaPath))
	hash := h.Sum(nil)

	fileName := fmt.Sprintf("%x%s", hash, cacheFileSuffix)
	globalCachePath := path.Join(m.config.CacheDir, fileName)
	if _, err := os.Stat(globalCachePath); err == nil {
		m.logger.Info().Str("path", globalCachePath).Msg("media global cache hit")
		return os.ReadFile(globalCachePath)
	}

	return nil, os.ErrNotExist
}

func (m *ManagerCtx) saveLocalCacheData(data []byte) error {
	localCachePath := m.config.MediaPath + cacheFileSuffix
	return os.WriteFile(localCachePath, data, 0755)
}

func (m *ManagerCtx) saveGlobalCacheData(data []byte) error {
	h := sha1.New()
	h.Write([]byte(m.config.MediaPath))
	hash := h.Sum(nil)

	fileName := fmt.Sprintf("%x%s", hash, cacheFileSuffix)
	globalCachePath := path.Join(m.config.CacheDir, fileName)
	return os.WriteFile(globalCachePath, data, 0755)
}
