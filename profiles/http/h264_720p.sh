#!/bin/sh

exec ffmpeg -hide_banner -loglevel warning \
  -i "${1}" \
  -vf scale=w=1280:h=720:force_original_aspect_ratio=decrease \
    -c:a aac \
      -ar 48000 \
      -b:a 128k \
    -c:v h264 \
      -profile:v main \
      -b:v 2800k \
      -maxrate 2996k \
      -bufsize 4200k \
      -crf 20 \
      -sc_threshold 0 \
      -g 48 \
      -keyint_min 48 \
  -f mpegts -
