#!/bin/sh

export VW="1920"
export VH="1080"
export ABANDWIDTH="192k"
export VBANDWIDTH="5000k"
export VMAXRATE="5350k"
export VBUFSIZE="7500k"

"$(dirname "$0")"/../http_h264.sh "$1"
