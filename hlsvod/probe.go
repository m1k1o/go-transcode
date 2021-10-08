package hlsvod

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
	"time"
)

type ProbeVideoData struct {
	Width      int
	Height     int
	Duration   time.Duration
	PktPtsTime []float64
}

func ProbeVideo(ctx context.Context, ffprobeBinary string, inputFilePath string) (*ProbeVideoData, error) {
	args := []string{
		"-v", "error", // Hide debug information
		"-ignore_chapters", "1",

		// video
		"-skip_frame", "nokey",
		"-show_entries", "frame=pkt_pts_time", // List all I frames
		"-show_entries", "format=duration",
		"-show_entries", "stream=duration,width,height",
		"-select_streams", "v", // Video stream only, we're not interested in audio

		"-of", "json",
		inputFilePath,
	}

	cmd := exec.CommandContext(ctx, ffprobeBinary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// TODO: Handle stderr output.
		log.Println(stderr.String())

		return nil, err
	}

	out := struct {
		Frames []struct {
			PktPtsTime string `json:"pkt_pts_time"`
		} `json:"frames"`
		Streams []struct {
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			Duration string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{}

	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, err
	}

	var duration time.Duration
	if out.Streams[0].Duration != "" {
		duration, err = time.ParseDuration(out.Streams[0].Duration + "s")
		if err != nil {
			return nil, err
		}
	}
	if out.Format.Duration != "" {
		duration, err = time.ParseDuration(out.Format.Duration + "s")
		if err != nil {
			return nil, err
		}
	}

	data := ProbeVideoData{
		Width:    out.Streams[0].Width,
		Height:   out.Streams[0].Height,
		Duration: duration,
	}

	for _, frame := range out.Frames {
		pktPtsTime, err := strconv.ParseFloat(frame.PktPtsTime, 64)
		if err != nil {
			return nil, err
		}

		data.PktPtsTime = append(data.PktPtsTime, pktPtsTime)
	}

	return &data, nil
}

type ProbeAudioData struct {
	Duration time.Duration
	BitRate  float64
}

func ProbeAudio(ctx context.Context, ffprobeBinary string, inputFilePath string) (*ProbeAudioData, error) {
	args := []string{
		"-v", "error", // Hide debug information
		"-ignore_chapters", "1",

		// audio
		"-show_entries", "stream=duration,bit_rate",
		"-select_streams", "a", // Audio stream only, we're not interested in video

		"-of", "json",
		inputFilePath,
	}

	cmd := exec.CommandContext(ctx, ffprobeBinary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// TODO: Handle stderr output.
		log.Println(stderr.String())

		return nil, err
	}

	out := struct {
		Streams []struct {
			BitRate  string `json:"bit_rate"`
			Duration string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}{}

	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, err
	}

	var duration time.Duration
	if out.Streams[0].Duration != "" {
		duration, err = time.ParseDuration(out.Streams[0].Duration + "s")
		if err != nil {
			return nil, err
		}
	}
	if out.Format.Duration != "" {
		duration, err = time.ParseDuration(out.Format.Duration + "s")
		if err != nil {
			return nil, err
		}
	}

	bitRate, err := strconv.ParseFloat(out.Streams[0].BitRate, 64)
	if err != nil {
		return nil, err
	}

	return &ProbeAudioData{
		Duration: duration,
		BitRate:  bitRate,
	}, nil
}
