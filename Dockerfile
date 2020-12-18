FROM golang:1.15-buster

WORKDIR /app

#
# install dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN set -eux; apt-get update; \
    apt-get install -y --no-install-recommends ffmpeg;
    #
    # clean up
    apt-get clean -y; \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/*

#
# build server
COPY . .

RUN go get -v -t -d .; \
    ./build

ENV TRANSCODE_BIND=:8080

ENTRYPOINT [ "bin/transcode" ]
CMD [ "serve" ]
