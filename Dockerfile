#
# STAGE 1: build executable binary
#
FROM golang:1.17-alpine as builder
WORKDIR /app

#
# build server
COPY . .
RUN go get -v -t -d .; \
    CGO_ENABLED=0 go build -o go-transcode

#
# STAGE 2: build a small image
#
# Since 3.18, alpine uses ffmpeg 6 which handles segments differently, see: https://github.com/m1k1o/go-transcode/issues/57
FROM alpine:3.17
WORKDIR /app

#
# install dependencies
RUN apk add --no-cache bash ffmpeg

#
# optional: install vdpau dependencies
ARG VDPAU="0"
RUN if [ "$VDPAU" = "1" ]; then \
        echo "https://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories; \
        apk update; \
        apk add --no-cache bash ffmpeg libva-utils libva-vdpau-driver libva-intel-driver intel-media-driver mesa-va-gallium; \
    fi

COPY --from=builder /app/go-transcode go-transcode
COPY profiles profiles

EXPOSE 8080
ENV TRANSCODE_BIND=:8080

ENTRYPOINT [ "./go-transcode" ]
CMD [ "serve" ]
