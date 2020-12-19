#!/bin/sh

ffmpeg -hide_banner -loglevel warning \
  -i "${1}" \
  -vf scale=w=1920:h=1080:force_original_aspect_ratio=decrease \
    -c:a aac \
      -ar 48000 \
      -b:a 192k \
    -c:v h264 \
      -profile:v main \
      -b:v 5000k \
      -maxrate 5350k \
      -bufsize 7500k \
      -crf 20 \
      -sc_threshold 0 \
      -g 48 \
      -keyint_min 48 \
  -f mpegts -
