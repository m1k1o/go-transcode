package hlsvod

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

/**
 * Calculate the timestamps to segment the video at.
 * Returns all segments endpoints, including video starting time (0) and end time.
 *
 * - Use keyframes (i.e. I-frame) as much as possible.
 * - For each key frame, if it's over (maxSegmentLength) seconds since the last keyframe, insert a breakpoint between them in an evenly,
 *   such that the breakpoint distance is <= (segmentLength) seconds (per https://bitmovin.com/mpeg-dash-hls-segment-length/).
 *   Example:
 *    segmentLength = 3.5
 *    key frame at 20.00 and 31.00, split at 22.75, 25.5, 28.25.
 * - If the duration between two key frames is smaller than (minSegmentLength) seconds, ignore the existance of the second key frame.
 *
 * This guarantees that all segments are between the duration (minSegmentLength) seconds and (maxSegmentLength) seconds.
 */
func convertToSegments(rawTimeList []float64, duration time.Duration, segmentLength float64, segmentOffset float64) []float64 {
	durationSec := duration.Seconds()
	minSegmentLength := segmentLength - segmentOffset
	maxSegmentLength := segmentLength + segmentOffset

	timeList := append(rawTimeList, durationSec)
	segmentStartTimes := []float64{0}

	lastTime := float64(0)
	for _, time := range timeList {
		// skip it regardless
		if time-lastTime < minSegmentLength {
			continue
		}

		// use it as-is
		if time-lastTime > minSegmentLength && time-lastTime < maxSegmentLength {
			lastTime = time
			segmentStartTimes = append(segmentStartTimes, lastTime)
			continue
		}

		// count segments between current and last time
		numOfSegmentsNeeded := math.Round((time - lastTime) / segmentLength)
		durationOfEach := (time - lastTime) / numOfSegmentsNeeded
		for i := 1; i < int(numOfSegmentsNeeded); i++ {
			lastTime += durationOfEach
			segmentStartTimes = append(segmentStartTimes, lastTime)
		}

		// use time directly instead of setting in the loop so we won't lose accuracy due to float point precision limit
		lastTime = time
		segmentStartTimes = append(segmentStartTimes, lastTime)
	}

	// would be equal to duration unless the skip branch is executed for the last segment, which is fixed below
	if len(segmentStartTimes) > 1 {
		// remove last segment start time
		segmentStartTimes = segmentStartTimes[:len(segmentStartTimes)-1]

		lastSegmentLength := durationSec - lastTime
		if lastSegmentLength > maxSegmentLength {
			segmentStartTimes = append(segmentStartTimes, durationSec-lastSegmentLength/2)
		}
	}

	return append(segmentStartTimes, durationSec)
}

func StreamsPlaylist(profiles map[string]VideoProfile, segmentNameFmt string) string {
	layers := []struct {
		Bitrate int
		Entries []string
	}{}

	for name, profile := range profiles {
		layers = append(layers, struct {
			Bitrate int
			Entries []string
		}{
			profile.Bitrate,
			[]string{
				fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,NAME=%s", profile.Bitrate, profile.Width, profile.Height, name),
				fmt.Sprintf(segmentNameFmt, name),
			},
		})
	}

	// sort by bitrate
	sort.Slice(layers, func(i, j int) bool {
		return layers[i].Bitrate < layers[j].Bitrate
	})

	// playlist prefix
	playlist := []string{"#EXTM3U"}

	// playlist segments
	for _, profile := range layers {
		playlist = append(playlist, profile.Entries...)
	}

	// join with newlines
	return strings.Join(playlist, "\n")
}
