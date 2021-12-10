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
FROM alpine
WORKDIR /app

#
# install dependencies
RUN apk add --no-cache bash ffmpeg

COPY --from=builder /app/go-transcode go-transcode
COPY profiles profiles

EXPOSE 8080
ENV TRANSCODE_BIND=:8080

ENTRYPOINT [ "./go-transcode" ]
CMD [ "serve" ]
