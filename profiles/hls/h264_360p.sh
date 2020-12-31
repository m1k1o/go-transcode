#!/bin/sh

ffmpeg -hide_banner -loglevel warning \
  -i "${1}" \
  -map 0:v:0 -map 0:a:0 \
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
  -f hls \
    -hls_time 2 \
    -hls_list_size 5 \
    -hls_wrap 10 \
    -hls_delete_threshold 1 \
    -hls_flags delete_segments \
    -hls_start_number_source datetime \
    -hls_segment_filename "live_%03d.ts" -
