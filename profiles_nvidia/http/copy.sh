#!/bin/bash

source "$(dirname "$0")/../.helpers.sh"

ffmpeg -hide_banner -loglevel warning \
  -hwaccel_output_format cuda \
  -c:v "$(cuvid_codec "${1}")" \
  -i "${1}" \
  -c:a copy \
  -c:v copy \
  -f mpegts -
