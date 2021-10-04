package hlsvod

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type TranscodeConfig struct {
	InputFilePath string
	OutputDirPath string
	SegmentPrefix string

	SegmentTimes []float64
	VideoProfile *VideoProfile
	AudioProfile *AudioProfile
}

type VideoProfile struct {
	Width   int
	Height  int
	Bitrate int // in kilobytes
}

type AudioProfile struct {
	Bitrate int // in kilobytes
}

// returns a channel, that delivers name of the segments as they are encoded
func TranscodeSegments(ctx context.Context, ffmpegBinary string, config TranscodeConfig) (chan string, error) {
	totalSegments := len(config.SegmentTimes)

	// set time bountary
	var startAt, endAt float64
	if totalSegments > 0 {
		startAt = config.SegmentTimes[0]
		endAt = config.SegmentTimes[totalSegments-1]
	}

	// convet to comma separated segment times
	fmtSegTimes := []string{}
	for _, segmentTime := range config.SegmentTimes {
		fmtSegTimes = append(
			fmtSegTimes,
			fmt.Sprintf("%.6f,", segmentTime),
		)
	}
	commaSeparatedSegTimes := strings.Join(fmtSegTimes[:], ",")

	args := []string{
		"-loglevel", "warning",
		"-ignore_chapters", "1",
	}

	// Seek to start point. Note there is a bug(?) in ffmpeg: https://github.com/FFmpeg/FFmpeg/blob/fe964d80fec17f043763405f5804f397279d6b27/fftools/ffmpeg_opt.c#L1240
	// can possible set `seek_timestamp` to a negative value, which will cause `avformat_seek_file` to reject the input timestamp.
	// To prevent this, the first break point, which we know will be zero, will not be fed to `-ss`.
	if startAt > 0 {
		args = append(args, []string{
			"-ss", fmt.Sprintf("%.6f", startAt),
		}...)
	}

	// Input specs
	args = append(args, []string{
		"-i", config.InputFilePath, // Input file
		"-to", fmt.Sprintf("%.6f", endAt),
		"-copyts", // So the "-to" refers to the original TS
		"-force_key_frames", commaSeparatedSegTimes,
		"-sn", // No subtitles
	}...)

	// Video specs
	if config.VideoProfile != nil {
		profile := config.VideoProfile

		var scale string
		if profile.Width >= profile.Height {
			scale = fmt.Sprintf("scale=-2:%d", profile.Height)
		} else {
			scale = fmt.Sprintf("scale=%d:-2", profile.Width)
		}

		args = append(args, []string{
			"-vf", scale,
			"-c:v", "libx264",
			"-preset", "faster",
			"-profile:v", "high",
			"-level:v", "4.0",
			"-b:v", fmt.Sprintf("%dk", profile.Bitrate),
		}...)
	}

	// Audio specs
	if config.AudioProfile != nil {
		profile := config.AudioProfile

		args = append(args, []string{
			"-c:a", "aac",
			"-b:a", fmt.Sprintf("%dk", profile.Bitrate),
		}...)
	}

	// Segmenting specs
	args = append(args, []string{
		"-f", "segment",
		"-segment_time_delta", "0.2",
		"-segment_format", "mpegts",
		"-segment_times", commaSeparatedSegTimes,
		"-segment_start_number", fmt.Sprintf("%.6f", startAt),
		"-segment_list_type", "flat",
		"-segment_list", "pipe:1", // Output completed segments to stdout.
		fmt.Sprintf("%s-%%05d.ts", config.SegmentPrefix),
	}...)

	cmd := exec.CommandContext(ctx, ffmpegBinary, args...)
	cmd.Dir = config.OutputDirPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	segments := make(chan string, 1)

	go func() {
		defer close(segments)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			segments <- scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			log.Println("Error while reading FFmpeg stdout:", err)
		}
	}()

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Println("FFmpeg process exited with error:", err)
		} else {
			log.Println("FFmpeg process successfully finished.")
		}
	}()

	return segments, cmd.Start()
}
