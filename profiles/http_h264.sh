#!/usr/bin/env bash

export INPUT="$1"

if [[ "$VW" = "" ]]; then echo "Missing \$VW"; exit 1; fi
if [[ "$VH" = "" ]]; then echo "Missing \$VH"; exit 1; fi
if [[ "$ABANDWIDTH" = "" ]]; then echo "Missing \$ABANDWIDTH"; exit 1; fi
if [[ "$VBANDWIDTH" = "" ]]; then echo "Missing \$VBANDWIDTH"; exit 1; fi
if [[ "$VMAXRATE" = "" ]]; then echo "Missing \$VMAXRATE"; exit 1; fi
if [[ "$VBUFSIZE" = "" ]]; then echo "Missing \$VBUFSIZE"; exit 1; fi

source "$(dirname "$0")/.helpers.hwaccel_h264.sh"

if [ -z "$CV" ] || [ -z "$VF" ]; then
  echo "Using CPU encoding."

  VF="scale=w=$VW:h=$VH:force_original_aspect_ratio=decrease"
  CV="h264"
fi

exec ffmpeg -hide_banner -loglevel warning \
  $EXTRAPARAMS \
  -i "$INPUT" \
  -vf $VF \
    -c:a aac \
      -ar 48000 \
      -ac 2 \
      -b:a $ABANDWIDTH \
    -c:v $CV \
      -profile:v main \
      -b:v $VBANDWIDTH \
      -maxrate $VMAXRATE \
      -bufsize $VBUFSIZE \
      -crf 20 \
      -sc_threshold 0 \
      -g 48 \
      -keyint_min 48 \
  -f mpegts -
