package hlsvod

import (
	"fmt"
	"math"
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

		// count segments between current and last time
		numOfSegmentsNeeded := math.Ceil((time - lastTime) / segmentLength)
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
	// playlist prefix
	playlist := []string{"#EXTM3U"}

	// playlist segments
	for name, profile := range profiles {
		playlist = append(playlist,
			fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,NAME=%s", profile.Bitrate, profile.Width, profile.Height, name),
			fmt.Sprintf(segmentNameFmt, name),
		)
	}

	// join with newlines
	return strings.Join(playlist, "\n")
}
