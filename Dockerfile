FROM golang:1.17-bullseye

WORKDIR /app

#
# install dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN set -eux; apt update; \
    apt install -y --no-install-recommends ffmpeg; \
    #
    # clean up
    apt clean -y; \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/*

#
# build server
COPY . .

RUN go get -v -t -d .; \
    go build -o bin/go-transcode

ENV TRANSCODE_BIND=:8080

ENTRYPOINT [ "bin/go-transcode" ]
CMD [ "serve" ]
