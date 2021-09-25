#!/bin/sh

export VW="1280"
export VH="720"
export ABANDWIDTH="128k"
export VBANDWIDTH="2800k"
export VMAXRATE="2996k"
export VBUFSIZE="4200k"

"$(dirname "$0")"/../http_h264.sh "$1"
