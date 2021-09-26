#!/bin/sh

export VW="960"
export VH="540"
export ABANDWIDTH="128k"
export VBANDWIDTH="1800k"
export VMAXRATE="1800k"
export VBUFSIZE="3100k"

"$(dirname "$0")"/../http_h264.sh "$1"
