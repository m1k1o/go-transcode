function vaapi_codec {
    SUPPORTED_CODECS=$(vainfo 2>&1 | grep VAProfile | sed s/VAProfile// | sed "s/\(Simple\|Main\|High\|Advanced\|Baseline\|Scc\|Constrain\|Profile\).*//"| sed s/None.*// | sort | uniq | sed 's/\ //g')
    CODEC=$(ffprobe -hide_banner -loglevel panic -select_streams v:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 "$1" | head -1)

    if [[ ! " ${SUPPORTED_CODECS} " =~ " ${CODEC} " ]]; then
        echo "Unsupported codec ${CODEC}."
        return 1
    fi
    echo "${CODEC}_vaapi"
}
