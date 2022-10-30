package hlsvod

import (
	"testing"
	"time"
)

func Test_convertToSegments(t *testing.T) {
	t.Run("difference between entries cannot be outside defined boundaries", func(t *testing.T) {
		// length, offset
		segmentTimes := [][]float64{
			{3.5, 1.25},
			{10, 5},
			{50, 1},
			{20, 19},
			{1, 0.5},
		}

		// ...semgents, duration
		inputs := [][]float64{
			{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			{5, 55, 555},
			{1, 1, 1},
			{5, 1, 9},
			{10},
			{0, 10, 20},
			{1},
			{0},
		}

		for _, segmentTime := range segmentTimes {
			segmentLength := segmentTime[0]
			segmentOffset := segmentTime[1]
			for _, input := range inputs {
				duration := time.Duration(input[len(input)-1]) * time.Second
				input = input[:len(input)-1]
				results := convertToSegments(input, duration, segmentLength, segmentOffset)

				var lastEl float64
				for _, el := range results {
					if lastEl != 0 {
						// expect(el - lastEl).to.be.at.least(segmentLength - segmentOffset)
						if el-lastEl < segmentLength-segmentOffset {
							t.Errorf("convertToSegments(%v) = %v, want at least %v", input, el-lastEl, segmentLength-segmentOffset)
						}
						// expect(el - lastEl).to.be.at.most(segmentLength + segmentOffset)
						if el-lastEl > segmentLength+segmentOffset {
							t.Errorf("convertToSegments(%v) = %v, want at most %v", input, el-lastEl, segmentLength+segmentOffset)
						}
					}

					lastEl = el
				}
			}
		}
	})
}
