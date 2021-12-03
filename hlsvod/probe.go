package hlsvod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ProbeMediaData struct {
	FormatName []string
	Duration   time.Duration

	Video *ProbeVideoData
	Audio []ProbeAudioData
}

func ProbeMedia(ctx context.Context, ffprobeBinary string, inputFilePath string) (*ProbeMediaData, error) {
	args := []string{
		"-v", "error", // Hide debug information
		"-show_format",  // Show container information
		"-show_streams", // Show codec information
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
			CodecName string `json:"codec_name"`
			CodecType string `json:"codec_type"`
			Duration  string `json:"duration"`

			// For video streams.
			Width  int `json:"width"`
			Height int `json:"height"`

			// For audio streams.
			BitRate string `json:"bit_rate"`
		} `json:"streams"`
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
		} `json:"format"`
	}{}

	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, err
	}

	data := ProbeMediaData{}
	for _, stream := range out.Streams {
		var duration time.Duration
		if stream.Duration != "" {
			duration, err = time.ParseDuration(stream.Duration + "s")
			if err != nil {
				return nil, fmt.Errorf("unable to parse stream duration: %v", err)
			}
		}

		switch stream.CodecType {
		case "video":
			if data.Video != nil {
				log.Printf("found multiple video streams for %s\n", inputFilePath)
			}

			data.Video = &ProbeVideoData{
				Width:    stream.Width,
				Height:   stream.Height,
				Duration: duration,
			}
		case "audio":
			var bitRate float64
			if out.Streams[0].BitRate != "" {
				bitRate, err = strconv.ParseFloat(out.Streams[0].BitRate, 64)
				if err != nil {
					return nil, fmt.Errorf("unable to parse audio stream bitrate: %v", err)
				}
			}

			data.Audio = append(data.Audio, ProbeAudioData{
				BitRate:  bitRate,
				Duration: duration,
			})
		}
	}

	if out.Format.FormatName != "" {
		data.FormatName = strings.Split(out.Format.FormatName, ",")
	}

	if out.Format.Duration != "" {
		data.Duration, err = time.ParseDuration(out.Format.Duration + "s")
		if err != nil {
			return nil, fmt.Errorf("unable to parse format duration: %v", err)
		}
	}

	return &data, nil
}

type ProbeVideoData struct {
	Width      int
	Height     int
	Duration   time.Duration
	PktPtsTime []float64
}

func ProbeVideo(ctx context.Context, ffprobeBinary string, inputFilePath string) (*ProbeVideoData, error) {
	args := []string{
		"-v", "error", // Hide debug information

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
		if frame.PktPtsTime == "" {
			continue
		}

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
