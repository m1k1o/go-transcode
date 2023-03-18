#!/bin/sh

export VW="640"
export VH="360"
export ABANDWIDTH="96k"
export VBANDWIDTH="800k"
export VMAXRATE="856k"
export VBUFSIZE="1200k"

exec "$(dirname "$0")"/../hls_h264.sh "$1"
