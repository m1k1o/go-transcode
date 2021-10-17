package utils

import (
	"io"
	"net/http"
	"sync"
	"time"
)

type Cache struct {
	mu      sync.RWMutex
	chunks  [][]byte
	length  int
	closed  bool
	closeCh chan struct{}

	listeners   []func([]byte) (int, error)
	listenersMu sync.RWMutex

	Expires time.Time
}

func NewCache(expires time.Time) *Cache {
	return &Cache{
		closeCh: make(chan struct{}),
		Expires: expires,
	}
}

func (c *Cache) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, io.ErrClosedPipe
	}

	// copy chunk
	dst := make([]byte, len(p))
	n = copy(dst, p)
	c.chunks = append(c.chunks, dst)
	c.length += n
	c.mu.Unlock()

	// broadcast
	c.listenersMu.RLock()
	for _, listener := range c.listeners {
		_, _ = listener(p)
	}
	c.listenersMu.RUnlock()

	return
}

func (c *Cache) Close() error {
	c.mu.Lock()
	c.closed = true
	close(c.closeCh)
	c.mu.Unlock()

	c.listenersMu.Lock()
	c.listeners = nil
	c.listenersMu.Unlock()

	return nil
}

func (c *Cache) ServeHTTP(w http.ResponseWriter) {
	offset := 0
	index := 0

	for {
		c.mu.RLock()
		length, closed := c.length, c.closed
		c.mu.RUnlock()

		// if we have enough available data
		if offset < length {
			i, _ := w.Write(c.chunks[index])
			offset += i
			index++
			continue
		}

		// if stream is already closed
		if closed {
			return
		}

		// we don't have enough data but stream is not closed
		break
	}

	// add current writer to listeners
	c.listenersMu.Lock()
	c.listeners = append(c.listeners, w.Write)
	c.listenersMu.Unlock()

	// wait until it finishes
	<-c.closeCh
}
