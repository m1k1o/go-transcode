#! /usr/bin/env bash

# Number of seconds to wait for HTTP server to start
# Test fails after this timeout
TIMEOUT=1

go build

# Test default settings (:8080)
output="$(TRANSCODE_BIND= timeout --preserve-status $TIMEOUT ./go-transcode serve 2>&1 3>&1)"
if echo "$output" | grep ":8080" > /dev/null; then
	echo "Default settings work"
else
	echo "Default settings failed:"
	echo "$output"
fi

# Test env settings (:8889)
output="$(TRANSCODE_BIND=":8889" timeout --preserve-status $TIMEOUT ./go-transcode serve 2>&1)"
if echo "$output" | grep ":8889" > /dev/null; then
	echo "Env settings work"
else
	echo "Env settings failed:"
	echo "$output"
fi

# Test CLI settings (:8890)
# We also check that CLI settings have higher priority than Env
output="$(TRANSCODE_BIND=":8889" timeout --preserve-status $TIMEOUT ./go-transcode serve --bind ":8890" 2>&1)"
if echo "$output" | grep ":8890" > /dev/null; then
	echo "CLI settings work"
else
	echo "CLI settings failed:"
	echo "$output"
fi


