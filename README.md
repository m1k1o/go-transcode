# Go live HTTP on-demand transcoding
Transcoding is expensive and resource consuming operation on CPU and GPU. For big companies with thousands of customer it is essential, to have a dedicated 24/7 transconding servers. But we, single sporadic users of transcoding, need to have different approach. Transcoding should be done only when its output is really needed. This tool is trying to solve this problem by offering transcoding on demand.

This tool is indended to be used with live streams only. Seeking is not supported, yet.

## Config
Specify streams as object in yaml file.

### Streams
Create `streams.yaml` file, with your sreams:

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

## CPU Profiles
Profiles with CPU transcoding can be found in `profiles`:

* h264_360p
* h264_540p
* h264_720p
* h264_1080p

They are accessible via: `http://localhost:8080/cpu/<profile>/<stream-id>`

Profile names must match flowing regex: `^[0-9A-Za-z_-]+$`

## GPU Profiles
Profiles with GPU transcoding can be found in `profiles_nvidia`:

* h264_360p
* h264_540p
* h264_720p
* h264_1080p

They are accessible via: `http://localhost:8080/gpu/<profile>/<stream-id>`

Profile names must match flowing regex: `^[0-9A-Za-z_-]+$`

## Docker

### Build

```sh
docker build -t transcode .
```

### Run

```sh
docker run --rm \
  --name="transcode" \
  -p "8080:8080" \
  -v "${PWD}/streams.yaml:/app/streams.yaml" transcode
```

## Nvidia GPU support (docker)

You will need to have [nvidia-docker](https://github.com/NVIDIA/nvidia-docker) installed.

### Build

First, you need to have build previus container and extract binary file from it.

```sh
docker cp transcode:/app/bin/transcode ./bin
```

Then, build nvidia container.

```sh
docker build -t transcode_nvidia -f Dockerfile.nvidia .
```

### Run

```sh
docker run --rm --gpus=all \
  --name="transcode_nvidia" \
  -p "8080:8080" \
  -v "${PWD}/streams.yaml:/app/streams.yaml" transcode_nvidia
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
