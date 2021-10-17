package hlsproxy

import (
	"io"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type CacheWriter struct {
	mu     sync.RWMutex
	chunks [][]byte
	length int
	closed bool

	Expires time.Time
}

func (w *CacheWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, io.ErrClosedPipe
	}

	n = len(p)
	w.chunks = append(w.chunks, p)
	w.length += n

	//log.Info().Int("length", w.length).Int("n", n).Msg("write")
	return
}

func (w *CacheWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.closed = true

	//log.Info().Int("length", w.length).Msg("close")
	return nil
}

func (w *CacheWriter) Reader() *CacheReader {
	return &CacheReader{0, 0, w}
}

type CacheReader struct {
	index  int
	offset int

	writer *CacheWriter
}

func (r *CacheReader) Read(b []byte) (n int, err error) {
	r.writer.mu.RLock()
	length, closed := r.writer.length, r.writer.closed
	r.writer.mu.RUnlock()

	// if we reached end of available buffer data
	if r.offset >= length {
		// if stream is already closed
		if closed {
			return 0, io.EOF
		}

		// TODO: remove busy waiting
		for {
			r.writer.mu.RLock()
			length, closed = r.writer.length, r.writer.closed
			r.writer.mu.RUnlock()

			if r.offset < length {
				break
			}

			if closed {
				return 0, io.EOF
			}
		}
	}

	n = copy(b, r.writer.chunks[r.index])
	r.offset += n
	r.index++

	//log.Info().Int("offset", r.offset).Int("index", r.index).Int("length", length).Int("n", n).Msg("read")
	return
}

func (m *ManagerCtx) getFromCache(key string) (io.Reader, bool) {
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
	return entry.Reader(), true
}

func (m *ManagerCtx) saveToCache(key string, reader io.Reader, duration time.Duration) io.Reader {
	log.Info().Str("key", key).Msg("cache add ++++")

	cacheWriter := CacheWriter{
		Expires: time.Now().Add(duration),
	}

	// pipe reader to writer.
	go func() {
		_, err := io.Copy(&cacheWriter, reader)
		log.Err(err).Msg("copied to cache")
		cacheWriter.Close()
	}()

	m.cacheMu.Lock()
	m.cache[key] = &cacheWriter
	m.cacheMu.Unlock()

	return cacheWriter.Reader()
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
