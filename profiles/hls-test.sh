#!/bin/sh

exec ffmpeg -re -r 30 -f lavfi -i testsrc -vf scale=1280:960 -vcodec libx264 -profile:v baseline -pix_fmt yuv420p \
  -f hls \
    -hls_time 2 \
    -hls_list_size 5 \
    -hls_delete_threshold 1 \
    -hls_flags delete_segments \
    -hls_start_number_source datetime \
    -hls_segment_filename "live_%03d.ts" \
    -
