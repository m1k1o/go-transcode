#!/bin/sh

exec ffmpeg -hide_banner -loglevel warning \
  -i "${1}" \
  -map 0:v:0 -map 0:a:0 \
  -c:a copy \
  -c:v copy \
  -f hls \
    -hls_time 2 \
    -hls_list_size 5 \
    -hls_wrap 10 \
    -hls_delete_threshold 1 \
    -hls_flags delete_segments \
    -hls_start_number_source datetime \
    -hls_segment_filename "live_%03d.ts" -
