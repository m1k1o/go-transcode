#!/bin/sh

exec ffmpeg -hide_banner -loglevel warning \
  -i "${1}" \
  -c:a copy \
  -c:v copy \
  -f mpegts -
