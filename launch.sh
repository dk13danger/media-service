#!/bin/bash -e

SERVICE_DIR=${PWD}
SERVICE_BINARY=media-service.o
DEBUG_MODE=true
URL="http://localhost:8080"

download_single() {
    curl "${URL}/dl?url=${1}&md5=${2}"
}

get_statistic() {
    if [ -z "$1" ]; then
        curl "${URL}/st"
    else
        curl "${URL}/st?url=${1}&md5=${2}"
    fi
}

case "$1" in
    "run")
        export DEBUG_MODE=${DEBUG_MODE}
        ./media-service.o
        ;;
    "run-docker")
        docker run -it --rm --name media-service \
            -e DEBUG_MODE=${DEBUG_MODE} \
            -p 8080:8080 \
            -v ${SERVICE_DIR}/cfg/prod.yml:/etc/media-service/config.yml \
            -v ${SERVICE_DIR}/sys/media.db:/etc/media-service/media.db \
            media-service:latest
        ;;
    "test-web")
        get_statistic "" ""
        ;;
    "test-web-params")
        get_statistic "http://www.sample-videos.com/video/mp4/720/big_buck_bunny_720p_1mb.mp4" "d55bddf8d62910879ed9f605522149a8"
        ;;
    "test-light")
        download_single "http://www.sample-videos.com/video/mp4/720/big_buck_bunny_720p_1mb.mp4" "d55bddf8d62910879ed9f605522149a8"
        ;;
    "test-heavy")
        INVALID_HASH="c689c2d468f841a20116992032dc09ca"
        SMALL_HASH="c689"

        download_single "http://www.sample-videos.com/video/mp4/720/big_buck_bunny_720p_1mb.mp4" "d55bddf8d62910879ed9f605522149a8"
        download_single "http://www.sample-videos.com/video/mp4/720/big_buck_bunny_720p_2mb.mp4" "${SMALL_HASH}"
        download_single "http://www.sample-videos.com/video/mp4/720/big_buck_bunny_720p_3mb.mp4" "${INVALID_HASH}"
        ;;
    *)
        echo "Usage: $(basename $0) <build> | <run> | <run-docker> | <test-web> | <test-web-params> | <test-light> | <test-heavy>"
        exit 1
       ;;
esac
