
INIT_HW_DEVICE="$(ffmpeg -init_hw_device list 2> /dev/null)"

#
# Apple Silicon
#
VIDEOTOOLBOX="yes"

if echo "$INIT_HW_DEVICE" | grep "videotoolbox" > /dev/null; then
    echo "[OK] Videotoolbox is supported by ffmpeg" >&2
else
    echo "[ERR]Videotoolbox is not supported by ffmpeg" >&2
    VIDEOTOOLBOX="no"
fi

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

function cuvid_decoder_from_codec {
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

if vainfo > /dev/null; then
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

function vaapi_check_supported_codecs {
    SUPPORTED_CODECS=$(vainfo 2>&1 | grep VAProfile | sed s/VAProfile// | grep Enc | sed "s/\(Simple\|Main\|High\|Advanced\|Baseline\|Scc\|Constrain\|Profile\).*//"| sed s/None.*// | sort | uniq | sed 's/\ //g')
    return [[ ! " ${SUPPORTED_CODECS} " =~ " ${1} " ]]
}

#
# FFMPEG PARAMS
#

if [ "$CUDA_SUPPORTED" = "yes" ]; then
    AVAILABLE_DECODERS="$(ffmpeg -hide_banner -decoders | grep _cuvid | cut -d ' ' -f3 | xargs)"
    AVAILABLE_FILTERS="$(ffmpeg -hide_banner -filters | grep cuda | cut -d ' ' -f3 | xargs)"

    CUVID_DECODER="$(cuvid_decoder_from_codec "$INPUT")"
    if [ "$?" = "1" ]; then
        echo "[ERR] CUDA - Unsupported codec: ${CUVID_DECODER:-???}" >&2
    elif [[ ! " ${AVAILABLE_DECODERS} " =~ " ${CUVID_DECODER} " ]]; then
        echo "[ERR] CUDA - Unsupported decoder: ${CUVID_DECODER:-???}" >&2
    else
        echo "Using CUDA hardware, with decoder $CUVID_DECODER." >&2
        export EXTRAPARAMS="-hwaccel_output_format cuda -c:v $CUVID_DECODER"

        # Check if filters are available
        if [[ " ${AVAILABLE_FILTERS} " =~ " hwupload_cuda " ]] &&
            [[ " ${AVAILABLE_FILTERS} " =~ " yadif_cuda " ]] &&
            [[ " ${AVAILABLE_FILTERS} " =~ " scale_cuda " ]]; then
            echo "Using CUDA filters" >&2
            export VF="hwupload_cuda,yadif_cuda=0:-1:0,scale_cuda=$VW:$VH:interp_algo=super:force_original_aspect_ratio=decrease"
        else
            echo "CUDA filters are not available" >&2
            export VF="scale=$VW:$VH:force_original_aspect_ratio=decrease"
        fi

        export CV="h264_nvenc"
    fi
elif [ "$VAAPI_SUPPORTED" = "yes" ]; then
    vaapi_check_supported_codecs H264
    if [ "$?" = "1" ]; then
        echo "[ERR] VAAPI - Unsupported codec: H264" >&2
    else
        echo "Using VAAPI hardware." >&2
        export EXTRAPARAMS="-hwaccel vaapi -hwaccel_device $DRI_RENDER -hwaccel_output_format vaapi"
        export VF="scale_vaapi=w=$VW:h=$VH:force_original_aspect_ratio=decrease"
        export CV="h264_vaapi"
    fi
elif [ "$VIDEOTOOLBOX" = "yes" ]; then
    echo "Using Videotoolbox h264_videotoolbox." >&2
    export EXTRAOUTPUTPARAMS="-q:v 80"
    export VF="scale=w=$VW:h=$VH:force_original_aspect_ratio=decrease"
    export CV="h264_videotoolbox"
else
    echo "No hardware acceleration available." >&2
fi
