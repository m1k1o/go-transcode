package hlsvod

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

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
		if time-lastTime < maxSegmentLength {
			lastTime = time
			segmentStartTimes = append(segmentStartTimes, lastTime)
			continue
		}

		// create as many segments as possible with perfect size
		for (time - lastTime) > segmentLength {
			lastTime += segmentLength
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
	return strings.Join(playlist, "\n") + "\n"
}
