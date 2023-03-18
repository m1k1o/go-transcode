#!/bin/sh

export VW="960"
export VH="540"
export ABANDWIDTH="128k"
export VBANDWIDTH="1800k"
export VMAXRATE="1800k"
export VBUFSIZE="3100k"

exec "$(dirname "$0")"/../hls_h264.sh "$1"
