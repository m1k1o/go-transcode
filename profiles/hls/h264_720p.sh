#!/bin/sh

ffmpeg -hide_banner -loglevel warning \
  -i "${1}" \
  -map 0:v:0 -map 0:a:0 \
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
  -f hls \
    -hls_time 2 \
    -hls_list_size 5 \
    -hls_wrap 10 \
    -hls_delete_threshold 1 \
    -hls_flags delete_segments \
    -hls_start_number_source datetime \
    -hls_segment_filename "live_%03d.ts" -
