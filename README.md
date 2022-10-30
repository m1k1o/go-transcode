# go-transcode HTTP on-demand transcoding API

On demand transcoding of live sources and static files (with seeking).

## Why

Transcoding is expensive and resource consuming operation on CPU and GPU. For big companies with thousands of customers it is essential, to have a dedicated 24/7 transcoding servers which can store all the transcoded versions.

For the rest of us who don't have infinite resources and cannot have 3 times bigger media library because of transcoding, we should only transcode when it is needed. This tool is trying to solve this problem by offering transcoding on demand.

This feature is common in media centers (plex, jellyfin) but there was no simple transcoding server without all other media center features. Now there is one! go-transcode is simple and extensible, and will probably not add features unrelated to transcoding.

## Features

Sources:
- [x] Live streams
- [x] VOD (static files, basic support)
- [x] Any codec/container supported by ffmpeg

Live Outputs:
- [x] Basic MP4 over HTTP (h264+aac) : `http://go-transcode/[profile]/[stream-id]`
- [x] Basic HLS over HTTP (h264+aac) : `http://go-transcode/[profile]/[stream-id]/index.m3u8`
- [x] Demo HTML player (for HLS) : `http://go-transcode/[profile]/[stream-id]/play.html`
- [x] HLS proxy : `http://go-transcode/hlsproxy/[hls-proxy-id]/[original-request]`

VOD Outputs:
- [x] HLS master playlist (h264+aac) : `http://go-transcode/vod/[media-path]/index.m3u8`
- [x] HLS custom profile (h264+aac) : `http://go-transcode/vod/[media-path]/[profile].m3u8`

Features:
- [x] Seeking for static files (indexed vod files)
- [ ] Audio/Subtitles tracks
- [ ] Private mode (serve users authenticated by reverse proxy)

You can find examples in [docs](./docs).

## Config

Place your config file in `./config.yaml` (or `/etc/transcode/config.yaml`). The streams are defined like this:

```yaml
streams:
  <stream-id>: <stream-url>
```

Full configuration example:

```yaml
# allow debug outputs
debug: true

# mount debug pprof endpoint at /debug/pprof/
pprof: true

# bind server to IP:PORT (use :8888 for all connections)
bind: localhost:8888

# serve static files from this directory (optional)
static: /var/www/html

# X-Forwarded-For headers will be used to determine the client IP
proxy: true

# For live streaming
streams:
  cam: rtmp://localhost/live/cam
  ch1_hd: http://192.168.1.34:9981/stream/channelid/85
  ch2_hd: http://192.168.1.34:9981/stream/channelid/43

# For static files
vod:
  # Source, where are static files, that will be transcoded
  media-dir: ./media
  # Temporary transcode output directory, if empty, default tmp folder will be used
  transcode-dir: ./transcode
  # Available video profiles
  video-profiles:
    360p:
      width: 640 # px
      height: 360 # px
      bitrate: 800 # kbps
    540p:
      width: 960
      height: 540
      bitrate: 1800
    720p:
      width: 1280
      height: 720
      bitrate: 2800
    1080p:
      width: 1920
      height: 1080
      bitrate: 5000
  # Use video keyframes as existing reference for chunks split
  # Using this might cause long probing times in order to get
  # all keyframes - therefore they should be cached
  video-keyframes: false
  # Single audio profile used
  audio-profile:
    bitrate: 192 # kbps
  # If cache is enabled
  cache: true
  # If dir is empty, cache will be stored in the same directory as media source
  # If not empty, cache files will be saved to specified directory
  cache-dir: ./cache
  # OPTIONAL: Use custom ffmpeg & ffprobe binary paths
  ffmpeg-binary: ffmpeg
  ffprobe-binary: ffprobe

# For proxying HLS streams
hls-proxy:
  my_server: http://192.168.1.34:9981
```

## Transcoding profiles for live streams

go-transcode supports any formats that ffmpeg likes. We provide profiles out-of-the-box for h264+aac (mp4 container) for 360p, 540p, 720p and 1080p resolutions: `h264_360p`, `h264_540p`, `h264_720p` and `h264_1080p`. Profiles can have any name, but must match regex: `^[0-9A-Za-z_-]+$`

In these profile directories, actual profiles are located in `hls/` and `http/`, depending on the output format requested. The profiles scripts detect hardware support by running ffmpeg. No special config needed to use hardware acceleration.

## Install

Clone repository and build with go compiler:

```sh
$ git clone https://github.com/m1k1o/go-transcode
$ cd go-transcode
$ go build
$ ./go-transcode serve
3:58PM WRN preflight complete without config file debug=false
3:56PM INF starting main server service=main
3:56PM INF http listening on 127.0.0.1:8080 module=http
3:56PM INF serving streams from basedir /home/klahaha/go-transcode: map[] service=main
3:56PM INF main ready service=main
```

First line is warning and "serving streams" line says empty list (`map[]`) because we don't have config.yaml so there no stream configured. Make your config.yaml and try again.

## Docker

### Build

```sh
docker build -t go-transcode:latest .
```

### Run

```sh
docker run --rm -d \
  --name="go-transcode" \
  -p "8080:8080" \
  -v "${PWD}/config.yaml:/app/config.yaml" go-transcode:latest
```

## VAAPI Support (docker)

```sh
docker run --rm -d \
  --name="go-transcode" \
  --device=/dev/dri:/dev/dri \
  -p "8080:8080" \
  -v "${PWD}/config.yaml:/app/config.yaml" go-transcode:latest
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
  -v "${PWD}/config.yaml:/app/config.yaml" go-transcode-nvidia:latest
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
- `hls/`: process runner for HLS transcoding
- `hlsvod/`: process runner for HLS VOD transcoding (for static files)
- `internal/`: actual source code logic

*TODO: document different modules/packages and dependencies*

Other files/folders in the repositories are:

- `data/`: files used/served by go-transcode
- `dev/`: some docker helper scripts
- `profiles/`: the ffmpeg profiles for transcoding
- `tests/`: some tests for the project
- `Dockerfile`, `Dockerfile.nvidia` and `docker-compose.yaml`: for the docker lovers
- `god.mod` and `go.sum`: golang dependencies/modules tracking
- `LICENSE`: licensing information (Apache 2.0)
