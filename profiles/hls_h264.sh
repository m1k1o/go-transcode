#!/usr/bin/env bash

if [[ "$VW" = "" ]]; then echo "Missing \$VW"; exit 1; fi
if [[ "$VH" = "" ]]; then echo "Missing \$VH"; exit 1; fi
if [[ "$ABANDWIDTH" = "" ]]; then echo "Missing \$ABANDWIDTH"; exit 1; fi
if [[ "$VBANDWIDTH" = "" ]]; then echo "Missing \$VBANDWIDTH"; exit 1; fi
if [[ "$VMAXRATE" = "" ]]; then echo "Missing \$VMAXRATE"; exit 1; fi
if [[ "$VBUFSIZE" = "" ]]; then echo "Missing \$VBUFSIZE"; exit 1; fi

HWSUPPORT="$(ffmpeg -init_hw_device list 2> /dev/null)"

if echo $HWSUPPORT | grep "^vaapi" > /dev/null; then
	# TODO: vaapi support
	#source "$(dirname "$0")/../.helpers.vaapi.sh"
	echo "NOT using VAAPI hardware (CPU fallback)."
	VF="scale=w=$VW:h=$VH:force_original_aspect_ratio=decrease"
	CV="h264"
elif echo $HWSUPPORT | grep "^cuda" > /dev/null; then
	echo "Using CUDA hardware."
	source "$(dirname "$0")/.helpers.cuda.sh"
	INPUT="$(cuvid_codec "${1}")"
	# ffmpeg parameters
	EXTRAPARAMS="-hwaccel_output_format cuda -c:v "$INPUT""
	# TODO: Why no force_original_aspect_ratio here?
	VF="hwupload_cuda,yadif_cuda=0:-1:0,scale_npp=$VW:$VH:interp_algo=super"
	CV="h264_nvenc"
else
	echo "Using CPU hardware."
	VF="scale=w=$VW:h=$VH:force_original_aspect_ratio=decrease"
	CV="h264"
fi

exec ffmpeg -hide_banner -loglevel warning \
  -i "${1}" $EXTRAPARAMS \
  -map 0:v:0 -map 0:a:0 \
  -vf $VF \
    -c:a aac \
      -ar 48000 \
      -ac 2 \
      -b:a $ABANDWIDTH \
    -c:v $CV \
      -profile:v main \
      -force_key_frames "expr:gte(t,n_forced*1)" \
      -b:v $VBANDWIDTH \
      -maxrate $VMAXRATE \
      -bufsize $VBUFSIZE \
      -crf 20 \
      -sc_threshold 0 \
      -g 48 \
      -keyint_min 48 \
  -f hls \
    -hls_time 2 \
    -hls_list_size 5 \
    -hls_wrap 10 \
    -hls_delete_threshold 1 \
    -hls_flags delete_segments+second_level_segment_index \
    -hls_start_number_source datetime \
    -strftime 1 \
    -hls_segment_filename "live_%Y%m%d%H%M%S_%%03d.ts" -
