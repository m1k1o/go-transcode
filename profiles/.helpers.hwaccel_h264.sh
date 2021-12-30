
INIT_HW_DEVICE="$(ffmpeg -init_hw_device list 2> /dev/null)"

#
# CUDA
#
CUDA_SUPPORTED="yes"

if echo "$INIT_HW_DEVICE" | grep "cuda" > /dev/null; then
    echo "[OK] CUDA is supported by ffmpeg." >&2
else
    echo "[ERR] CUDA is not supported by ffmpeg." >&2
    CUDA_SUPPORTED="no"
fi

function cuvid_codec_to_encoder {
    SUPPORTED_CODECS="h264 hevc mjpeg mpeg1video mpeg2video mpeg4 vc1 vp8 vp9"
    CODEC=$(ffprobe -hide_banner -loglevel panic -select_streams v:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 "$1" | head -1)

    if [[ ! " ${SUPPORTED_CODECS} " =~ " ${CODEC} " ]]; then
        echo "${CODEC}"
        return 1
    fi

    if [[ "${CODEC}" == "mpeg1video" ]]; then
        echo "mpeg1_cuvid"
        return 0
    fi

    if [[ "${CODEC}" == "mpeg2video" ]]; then
        echo "mpeg2_cuvid"
        return 0
    fi

    echo "${CODEC}_cuvid"
}

#
# VAAPI
#
VAAPI_SUPPORTED="yes"

if echo "$INIT_HW_DEVICE" | grep "vaapi" > /dev/null; then
    echo "[OK] VAAPI is supported by ffmpeg." >&2
else
    echo "[ERR] VAAPI is not supported by ffmpeg." >&2
    VAAPI_SUPPORTED="no"
fi

if vainfo 2>&1 > /dev/null; then
    echo "[OK] vainfo command executed succesfully." >&2
else
    echo "[ERR] vainfo command not found." >&2
    VAAPI_SUPPORTED="no"
fi

DRI_RENDER="/dev/dri/renderD128"
if [ -e $DRI_RENDER ]; then
    echo "[OK] $DRI_RENDER exists." >&2
else
    echo "[ERR] $DRI_RENDER not found." >&2
    VAAPI_SUPPORTED="no"
fi

function vaapi_codec_to_encoder {
    SUPPORTED_CODECS=$(vainfo 2>&1 | grep VAProfile | sed s/VAProfile// | grep Enc | sed "s/\(Simple\|Main\|High\|Advanced\|Baseline\|Scc\|Constrain\|Profile\).*//"| sed s/None.*// | sort | uniq | sed 's/\ //g')
    CODEC=$(ffprobe -hide_banner -loglevel panic -select_streams v:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 "$1" | head -1)

    if [[ ! " ${SUPPORTED_CODECS} " =~ " ${CODEC} " ]]; then
        echo "${CODEC}"
        return 1
    fi

    echo "${CODEC}_vaapi"
}

#
# FFMPEG PARAMS
#

if [ "$CUDA_SUPPORTED" = "yes" ]; then
    AVAILABLE_ENCODERS="$(ffmpeg -hide_banner -encoders | grep _cuvid | cut -d ' ' -f3)"

    ENCODER="$(cuvid_codec_to_encoder "$INPUT")"
    if [ "$?" = "1" ]; then
        echo "[ERR] CUDA - Unsupported codec: ${ENCODER:-???}" >&2
    elif [[ ! " ${AVAILABLE_ENCODERS} " =~ " ${ENCODER} " ]]; then
        echo "[ERR] CUDA - Unsupported encoder: ${ENCODER:-???}" >&2
    else
        echo "Using CUDA hardware, codec $ENCODER." >&2
        export EXTRAPARAMS="-hwaccel_output_format cuda -c:v $ENCODER"
        # TODO: Why no force_original_aspect_ratio here?
        export VF="hwupload_cuda,yadif_cuda=0:-1:0,scale_npp=$VW:$VH:interp_algo=super"
        export CV="h264_nvenc"
    fi
elif [ "$VAAPI_SUPPORTED" = "yes" ]; then
    AVAILABLE_ENCODERS="$(ffmpeg -hide_banner -encoders | grep _vaapi | cut -d ' ' -f3)"

    ENCODER="$(vaapi_codec_to_encoder "$INPUT")"
    if [ "$?" = "1" ]; then
        echo "[ERR] VAAPI - Unsupported codec: ${ENCODER:-???}" >&2
    elif [[ ! " ${AVAILABLE_ENCODERS} " =~ " ${ENCODER} " ]]; then
        echo "[ERR] VAAPI - Unsupported encoder: ${ENCODER:-???}" >&2
    else
        echo "Using using VAAPI hardware, codec $ENCODER." >&2
        export EXTRAPARAMS="-hwaccel vaapi -hwaccel_device $DRI_RENDER -hwaccel_output_format vaapi -c:v $ENCODER"
        export VF="scale_vaapi=w=$VW:h=$VH:force_original_aspect_ratio=decrease"
        export CV="h264_vaapi"
    fi
else
    echo "No hardware acceleration available." >&2
fi
