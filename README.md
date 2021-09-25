# go-transcode HTTP on-demand transcoding API

## Why

Transcoding is expensive and resource consuming operation on CPU and GPU. For big companies with thousands of customers it is essential, to have a dedicated 24/7 transcoding servers which can store all the transcoded versions.

For the rest of us who don't have infinite resources and cannot have 3 times bigger media library because of transcoding, we should only transcode when it is needed. This tool is trying to solve this problem by offering transcoding on demand.

This feature is common in media centers (plex, jellyfin) but there was no simple transcoding server without all other media center features. Now there is one! go-transcode is simple and extensible, and will probably not add features unrelated to transcoding.

## Features

Sources:
- [x] Live streams
- [ ] Static files (basic support)
- [x] Any codec/container supported by ffmpeg

Outputs:
- [x] Basic MP4 over HTTP (h264+aac) : `http://go-transcode/[profile]/[stream-id]`
- [x] Basic HLS over HTTP (h264+aac) : `http://go-transcode/[profile]/[stream-id]/index.m3u8`
- [x] Demo HTML player (for HLS) : `http://go-transcode/[profile]/[stream-id]/play.html`

Features:
- [ ] Seeking for static files (index)
- [ ] Audio/Subtitles tracks
- [ ] Private mode (serve users authenticated by reverse proxy)

## Config

Place your config file in `./transcode.yml` (or `/etc/transcode/transcode.yml`). The streams are defined like this:

```yaml
streams:
  <stream-id>: <stream-url>
```

Full configuration example:

```yaml
# allow debug outputs
debug: true

# bind server to IP:PORT (use :8888 for all connections)
bind: localhost:8888

# serve static files from this directory (optional)
static: /var/www/html

# TODO: issue #4
proxy: true

streams:
  cam: rtmp://localhost/live/cam
  ch1_hd: http://192.168.1.34:9981/stream/channelid/85
  ch2_hd: http://192.168.1.34:9981/stream/channelid/43
```

## Transcoding profiles

go-transcode supports any formats that ffmpeg likes. We provide profiles out-of-the-box for h264+aac (mp4 container) for 360p, 540p, 720p and 1080p resolutions: `h264_360p`, `h264_540p`, `h264_720p` and `h264_1080p`. Profiles can have any name, but must match regex: `^[0-9A-Za-z_-]+$`

We provide two different profiles directories:

- profiles/default for CPU transcoding
- profiles/nvidia for NVENC support (proprietary Nvidia driver)

In these profile directories, actual profiles are located in `hls/` and `http/`, depending on the output format requested.

## Docker

*TODO: outdated docker section*

### Build

```sh
docker build -t go-transcode:latest .
```

### Run

```sh
docker run --rm -d \
  --name="go-transcode" \
  -p "8080:8080" \
  -v "${PWD}/streams.yaml:/app/streams.yaml" go-transcode:latest
```

## Nvidia GPU support (docker)

You will need to have [nvidia-docker](https://github.com/NVIDIA/nvidia-docker) installed.

### Build

First, you need to build previous container. Then, build Nvidia container.

```sh
docker build --build-arg "TRANSCODE_IMAGE=go-transcode:latest" -t go-transcode-nvidia:latest -f Dockerfile.nvidia .
```

### Run

```sh
docker run --rm -d \
  --gpus=all \
  --name="go-transcode-nvidia" \
  -p "8080:8080" \
  -v "${PWD}/streams.yaml:/app/streams.yaml" go-transcode-nvidia:latest
```

### Supported inputs

Input codec will be automatically determined from given stream. Please check your graphic card's supported codec and maximum concurrent sessions [here](https://developer.nvidia.com/video-encode-decode-gpu-support-matrix).

| Codec      | CUVID       | Codec Name                                |
| ---------- | ----------- | ----------------------------------------- |
| h264       | h264_cuvid  | H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 |
| hevc       | hevc_cuvid  | H.265 / HEVC                              |
| mjpeg      | mjpeg_cuvid | Motion JPEG                               |
| mpeg1video | mpeg1_cuvid | MPEG-1 video                              |
| mpeg2video | mpeg2_cuvid | MPEG-2 video                              |
| mpeg4      | mpeg4_cuvid | MPEG-4 part 2                             |
| vc1        | vc1_cuvid   | SMPTE VC-1                                |
| vp8        | vp8_cuvid   | On2 VP8                                   |
| vp9        | vp9_cuvid   | Google VP9                                |

## Alternatives

- [nginx-vod-module](https://github.com/kaltura/nginx-vod-module): Only supports MP4 sources.
- [tvheadend](https://tvheadend.org/): Intended for various live sources (IPTV or DVB), not media library - although it can record TV. Supports Nvidia acceleration, but it is hard to compile.
- [jellyfin](https://github.com/jellyfin/jellyfin): Supports live TV sources, although does not work realiably. Cannot run standalone transcoding service (without media library).
- Any suggestions?

## Contribute

Join us in the [Matrix space](https://matrix.to/#/#go-transcode:proxychat.net) (or the [#go-transcode-general](https://matrix.to/#/#go-transcode-general:proxychat.net) room directly) or [via XMPP bridge](xmpp:#go-transcode-general#proxychat.net@matrix.org).

## Architecture

The source code is in the following files/folders:

- `cmd/` and `main.go`: source for the command-line interface
- `internal/`: actual source code logic
- `hls/`: process runner for HLS transcoding

*TODO: document different modules/packages and dependencies*

Other files/folders in the repositories are:

- `data/`: files used/served by go-transcode
- `profiles/ and profiles_nvidia/`: the ffmpeg profiles for transcoding
- `dev/`: some docker helper scripts
- `tests/`: some tests for the project
- `god.mod` and `go.sum`: golang dependencies/modules tracking
- `Dockerfile`, `Dockerfile.nvidia` and `docker-compose.yaml`: for the docker lovers
- `LICENSE`: licensing information (Apache 2.0)
