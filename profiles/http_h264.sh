#!/usr/bin/env bash

if [[ "$VW" = "" ]]; then echo "Missing \$VW"; exit 1; fi
if [[ "$VH" = "" ]]; then echo "Missing \$VH"; exit 1; fi
if [[ "$ABANDWIDTH" = "" ]]; then echo "Missing \$ABANDWIDTH"; exit 1; fi
if [[ "$VBANDWIDTH" = "" ]]; then echo "Missing \$VBANDWIDTH"; exit 1; fi
if [[ "$VMAXRATE" = "" ]]; then echo "Missing \$VMAXRATE"; exit 1; fi
if [[ "$VBUFSIZE" = "" ]]; then echo "Missing \$VBUFSIZE"; exit 1; fi
#run vainfo to get exit code as output saved using $?
vainfo 2>&1 > /dev/null
#store exit code as support 0 means vaapi works 1 means software if cuda not detected
VAAPISUPPORT=$($?)
CUDASUPPORT="$(ffmpeg -init_hw_device list 2> /dev/null)"

if echo $CUDASUPPORT | grep "cuda" > /dev/null; then
        echo "Using CUDA hardware."
        source "$(dirname "$0")/.helpers.cuda.sh"
        INPUT="$(cuvid_codec "${1}")"
        # ffmpeg parameters
        EXTRAPARAMS="-hwaccel_output_format cuda -c:v "$INPUT""
        # TODO: Why no force_original_aspect_ratio here?
        VF="hwupload_cuda,yadif_cuda=0:-1:0,scale_npp=$VW:$VH:interp_algo=super"
        CV="h264_nvenc"
fi
if [ "$VAAPISUPPORT" -eq "0" ]; then
        # TODO: vaapi support
        source "$(dirname "$0")/.helpers.vaapi.sh"
        #EXTRAPRAMAS="-hwaccel_output_format vaapi"
        HWDECODE="-hwaccel vaapi -hwaccel_device /dev/dri/renderD128 -hwaccel_output_format vaapi"
        echo "Using using VAAPI hardware (CPU fallback)."
        VF="scale_vaapi=w=$VW:h=$VH:force_original_aspect_ratio=decrease"
        CV="h264_vaapi"
else
        echo "Using CPU hardware."
        VF="scale=w=$VW:h=$VH:force_original_aspect_ratio=decrease"
        CV="h264"
fi

exec ffmpeg -hide_banner -loglevel warning \
   $HWDECODE \
  -i "${1}" $EXTRAPARAMS \
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
