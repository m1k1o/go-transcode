function cuvid_codec {
    SUPPORTED_CODECS="h264 hevc mjpeg mpeg1video mpeg2video mpeg4 vc1 vp8 vp9"
    CODEC=$(ffprobe -hide_banner -loglevel panic -select_streams v:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 "$1" | head -1)

    if [[ ! " ${SUPPORTED_CODECS} " =~ " ${CODEC} " ]]; then
        echo "Unsupported codec ${CODEC}."
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
