#!/bin/bash

source "$(dirname "$0")/../.helpers.sh"

exec ffmpeg -hide_banner -loglevel warning \
  -hwaccel_output_format cuda \
  -c:v "$(cuvid_codec "${1}")" \
  -i "${1}" \
  -vf hwupload_cuda,yadif_cuda=0:-1:0,scale_npp=640:360:interp_algo=super \
    -c:a aac \
      -ar 48000 \
      -b:a 96k \
    -c:v h264_nvenc \
      -profile:v main \
      -b:v 800k \
      -maxrate 856k \
      -bufsize 1200k \
      -crf 20 \
      -sc_threshold 0 \
      -g 48 \
      -keyint_min 48 \
  -f mpegts -
