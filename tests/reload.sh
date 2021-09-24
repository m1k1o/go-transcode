#! /usr/bin/env bash

go build || exit 1

if [ ! -f $1 ]; then 
	echo "Please give test video first argument"
	exit 1
fi

tmpfile=$(mktemp --suffix .yml)
log=$(mktemp)

BASE_PORT=8888
INCREMENT=1

port=$BASE_PORT
isfree=$(netstat -taln | grep $port)

while [[ -n "$isfree" ]]; do
    port=$[port+INCREMENT]
    isfree=$(netstat -taln | grep $port)
done

if [[ -n "$isfree" ]]; then
	echo "Could not find free port for test"
	exit 1
fi

baseurl="http://localhost:$port"

echo "Using port: $port. Logging to $log"

echo -e "streams:\n  test: $1" > $tmpfile

./go-transcode serve --config $tmpfile --bind :$port >> $log 2>&1 &
pid=$!

output="$(curl -o /dev/null -s -I -XGET -w "%{http_code}" $baseurl/h264_720p/test)"
if [[ "$output" != "200" ]]; then
	echo "$output"
	echo "Serve 1 failed"
	exit 1
else
	echo "Serve 1 success"
fi

output="$(curl -o /dev/null -s -I -XGET -w "%{http_code}" $baseurl/h264_720p/test2)"
if [[ "$output" != "404" ]]; then
	echo "Serve 2 failed"
	exit 1
else
	echo "Serve 2 success"
fi

# Change config and try again test2
echo -e "streams:\n  test2: $1" > $tmpfile
sleep 1

output="$(curl -o /dev/null -s -I -XGET -w "%{http_code}" $baseurl/h264_720p/test2)"
if [[ "$output" != "200" ]]; then
	echo "Serve 3 failed"
	exit 1
else
	echo "Serve 3 success"
fi

output="$(curl -o /dev/null -s -I -XGET -w "%{http_code}" $baseurl/h264_720p/test)"
if [[ "$output" != "404" ]]; then
	echo "Serve 4 failed"
	exit 1
else
	echo "Serve 4 success"
fi


