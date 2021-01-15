# Go live HTTP on-demand transcoding
Transcoding is expensive and resource consuming operation on CPU and GPU. For big companies with thousands of customers it is essential, to have a dedicated 24/7 transcoding servers. But we, single sporadic users of transcoding, need to have different approach. Transcoding should be done only when its output is really needed. This tool is trying to solve this problem by offering transcoding on demand.

This tool is intended to be used with live streams only. Seeking is not supported, yet.

## Config
Specify streams as object in yaml file.

### Streams
Create `streams.yaml` file, with your streams:

```yaml
streams:
  <stream-id>: <stream-url>
```

Example:
```yaml
streams:
  cam: rtmp://localhost/live/cam
  ch1_hd: http://192.168.1.34:9981/stream/channelid/85
  ch2_hd: http://192.168.1.34:9981/stream/channelid/43
```

HTTP streaming is accessible via:
- `http://localhost:8080/<profile>/<stream-id>`

HLS is accessible via:
- `http://localhost:8080/<profile>/<stream-id>/index.m3u8`
- `http://localhost:8080/<profile>/<stream-id>/play.html`

## CPU Profiles
Profiles (HTTP and HLS) with CPU transcoding can be found in `profiles`:

* h264_360p
* h264_540p
* h264_720p
* h264_1080p

Profile names must match flowing regex: `^[0-9A-Za-z_-]+$`

## GPU Profiles
Profiles (HTTP and HLS) with GPU transcoding can be found in `profiles_nvidia`:

* h264_360p
* h264_540p
* h264_720p
* h264_1080p

Profile names must match flowing regex: `^[0-9A-Za-z_-]+$`

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
