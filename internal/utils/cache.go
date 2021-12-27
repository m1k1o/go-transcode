package utils

import (
	"io"
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

	expires time.Time
}

func NewCache(expires time.Time) *Cache {
	return &Cache{
		closeCh: make(chan struct{}),
		expires: expires,
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

func (c *Cache) CopyTo(w io.Writer) error {
	offset, index := 0, 0

	for {
		c.mu.RLock()
		length, closed := c.length, c.closed
		c.mu.RUnlock()

		// if we have enough available data
		if offset < length {
			c.mu.RLock()
			chunk := c.chunks[index]
			c.mu.RUnlock()

			i, err := w.Write(chunk)
			if err != nil {
				return err
			}

			offset += i
			index++
			continue
		}

		// if stream is already closed
		if closed {
			var err error
			if closer, ok := w.(io.WriteCloser); ok {
				err = closer.Close()
			}
			return err
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
	return nil
}

func (c *Cache) Expired() bool {
	return time.Now().After(c.expires)
}
