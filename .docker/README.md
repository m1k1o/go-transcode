# Docker

## Build

```sh
./build go-transcode:latest
```

## Run

```sh
docker run --rm -d \
  --name="go-transcode" \
  -p "8080:8080" \
  -v "${PWD}/config.yaml:/app/config.yaml" go-transcode:latest
```

# Nvidia GPU support

You will need to have [nvidia-docker](https://github.com/NVIDIA/nvidia-docker) installed.

## Build

First, you need to build previous container. Then, build Nvidia container.

```sh
docker build --build-arg "TRANSCODE_IMAGE=go-transcode:latest" -t go-transcode-nvidia:latest -f Dockerfile.nvidia ..
```

## Run

```sh
docker run --rm -d \
  --gpus=all \
  --name="go-transcode-nvidia" \
  -p "8080:8080" \
  -v "${PWD}/config.yaml:/app/config.yaml" go-transcode-nvidia:latest
```

## Supported inputs

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
