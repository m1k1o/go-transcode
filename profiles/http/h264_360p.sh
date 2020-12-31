#!/bin/sh

ffmpeg -hide_banner -loglevel warning \
  -i "${1}" \
  -vf scale=w=640:h=360:force_original_aspect_ratio=decrease \
    -c:a aac \
      -ar 48000 \
      -b:a 96k \
    -c:v h264 \
      -profile:v main \
      -b:v 800k \
      -maxrate 856k \
      -bufsize 1200k \
      -crf 20 \
      -sc_threshold 0 \
      -g 48 \
      -keyint_min 48 \
  -f mpegts -
