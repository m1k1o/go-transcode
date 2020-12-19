#!/bin/sh

cd "$(dirname "$0")"
source .helpers.sh

ffmpeg -hide_banner -loglevel warning \
  -hwaccel_output_format cuda \
  -c:v "$(cuvid_codec "${1}")" \
  -i "${1}" \
  -vf hwupload_cuda,yadif_cuda=0:-1:0,scale_npp=1280:720:interp_algo=super \
    -c:a aac \
      -ar 48000 \
      -b:a 128k \
    -c:v h264_nvenc \
      -profile:v main \
      -b:v 2800k \
      -maxrate 2996k \
      -bufsize 4200k \
      -crf 20 \
      -sc_threshold 0 \
      -g 48 \
      -keyint_min 48 \
  -f mpegts -
