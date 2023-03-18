#!/bin/sh

exec ffmpeg \
  -re -r 30 \
  -f lavfi -i testsrc \
  -vf scale=1280:960 \
  -c:v libx264 \
    -profile:v baseline \
    -pix_fmt yuv420p \
  -f mpegts \
  -
