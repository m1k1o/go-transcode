package hlsvod

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path"
)

func (m *ManagerCtx) getCacheData() ([]byte, error) {
	// check for local cache
	localCachePath := fmt.Sprintf("%s.go-transcode-cache", m.config.MediaPath)
	if _, err := os.Stat(localCachePath); err == nil {
		m.logger.Warn().Str("path", localCachePath).Msg("media local cache hit")
		return os.ReadFile(localCachePath)
	}

	// check for global cache
	h := sha1.New()
	h.Write([]byte(m.config.MediaPath))
	hash := h.Sum(nil)

	globalCachePath := path.Join(m.config.CacheDir, fmt.Sprintf("%s.go-transcode-cache", hash))
	if _, err := os.Stat(globalCachePath); err == nil {
		m.logger.Warn().Str("path", globalCachePath).Msg("media global cache hit")
		return os.ReadFile(globalCachePath)
	}

	return nil, os.ErrNotExist
}

func (m *ManagerCtx) saveLocalCacheData(data []byte) error {
	localCachePath := fmt.Sprintf("%s.go-transcode-cache", m.config.MediaPath)
	return os.WriteFile(localCachePath, data, 0755)
}

func (m *ManagerCtx) saveGlobalCacheData(data []byte) error {
	h := sha1.New()
	h.Write([]byte(m.config.MediaPath))
	hash := h.Sum(nil)

	globalCachePath := path.Join(m.config.CacheDir, fmt.Sprintf("%s.go-transcode-cache", hash))
	return os.WriteFile(globalCachePath, data, 0755)
}
