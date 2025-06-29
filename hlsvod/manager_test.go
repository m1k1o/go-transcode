package hlsvod

import (
	"context"
	"testing"
)

// TestTranscodeFromSegmentBufferSize ensures that transcodeFromSegment()
// queues exactly segmentBufferMax segments for transcoding. (This is
// a regression test, as the implementation previously only queued
// segmentBufferMax-1 segments.)
func TestTranscodeFromSegmentBufferSize(t *testing.T) {
	const bufferMax = 5

	// Prepare a ManagerCtx with exactly bufferMax segments.
	m := &ManagerCtx{
		breakpoints:      make([]float64, bufferMax+1),
		segmentBufferMax: bufferMax,
		segmentBufferMin: 3, // default value from constructor
		segments:         make(map[int]string),
		segmentQueue:     make(map[int]chan struct{}),
	}

	// Populate the dummy segments map so that len(m.segments) == bufferMax.
	for i := 0; i < bufferMax; i++ {
		m.segments[i] = ""
	}

	// Stub out the transcode function so the test doesn't invoke FFmpeg.
	origFn := transcodeSegmentsFn
	transcodeSegmentsFn = func(_ context.Context, _ string, _ TranscodeConfig) (chan string, error) {
		ch := make(chan string)
		close(ch)
		return ch, nil
	}
	defer func() { transcodeSegmentsFn = origFn }()

	// Execute the code under test.
	if err := m.transcodeFromSegment(0); err != nil {
		t.Fatalf("transcodeFromSegment returned error: %v", err)
	}

	// transcodeFromSegment should enqueue `bufferMax` segments
	if got := len(m.segmentQueue); got != bufferMax {
		t.Fatalf("expected %d queued segments, got %d", bufferMax, got)
	}
}
