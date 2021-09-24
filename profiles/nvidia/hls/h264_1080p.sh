#!/bin/bash

source "$(dirname "$0")/../.helpers.sh"

exec ffmpeg -hide_banner -loglevel warning \
  -hwaccel_output_format cuda \
  -c:v "$(cuvid_codec "${1}")" \
  -i "${1}" \
  -map 0:v:0 -map 0:a:0 \
  -vf hwupload_cuda,yadif_cuda=0:-1:0,scale_npp=1920:1080:interp_algo=super \
    -c:a aac \
      -ar 48000 \
      -b:a 192k \
    -c:v h264_nvenc \
      -profile:v main \
      -b:v 5000k \
      -maxrate 5350k \
      -bufsize 7500k \
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
