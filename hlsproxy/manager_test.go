package hlsproxy

import (
	"bytes"
	"io"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestPlaylistUrlWalk(t *testing.T) {
	type args struct {
		input   string
		replace func(string) string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple: absolute URL",
			args: args{
				input: `#EXTM3U
					#EXT-X-VERSION:3
					#EXT-X-STREAM-INF:BANDWIDTH=1000000,RESOLUTION=1280x720
					http://example.com/720p.m3u8
					#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=854x480
					http://example.com/480p.m3u8
					#EXT-X-STREAM-INF:BANDWIDTH=250000,RESOLUTION=640x360
					http://example.com/360p.m3u8?streamer=456
					#EXT-X-STREAM-INF:BANDWIDTH=125000,RESOLUTION=426x240
					http://example.com/240p.m3u8
				`,
				replace: func(s string) string { return "!!" + s + "!!" },
			},
			want: `#EXTM3U
				#EXT-X-VERSION:3
				#EXT-X-STREAM-INF:BANDWIDTH=1000000,RESOLUTION=1280x720
				!!http://example.com/720p.m3u8!!
				#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=854x480
				!!http://example.com/480p.m3u8!!
				#EXT-X-STREAM-INF:BANDWIDTH=250000,RESOLUTION=640x360
				!!http://example.com/360p.m3u8?streamer=456!!
				#EXT-X-STREAM-INF:BANDWIDTH=125000,RESOLUTION=426x240
				!!http://example.com/240p.m3u8!!
			`,
		},
		{
			name: "simple: relative URL",
			args: args{
				input: `#EXTM3U
					#EXT-X-VERSION:3
					#EXT-X-STREAM-INF:BANDWIDTH=1000000,RESOLUTION=1280x720
					/720p.m3u8
					#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=854x480
					/480p.m3u8
					#EXT-X-STREAM-INF:BANDWIDTH=250000,RESOLUTION=640x360
					/360p.m3u8?streamer=456
					#EXT-X-STREAM-INF:BANDWIDTH=125000,RESOLUTION=426x240
					/240p.m3u8
				`,
				replace: func(s string) string { return "http://example.com" + s },
			},
			want: `#EXTM3U
				#EXT-X-VERSION:3
				#EXT-X-STREAM-INF:BANDWIDTH=1000000,RESOLUTION=1280x720
				http://example.com/720p.m3u8
				#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=854x480
				http://example.com/480p.m3u8
				#EXT-X-STREAM-INF:BANDWIDTH=250000,RESOLUTION=640x360
				http://example.com/360p.m3u8?streamer=456
				#EXT-X-STREAM-INF:BANDWIDTH=125000,RESOLUTION=426x240
				http://example.com/240p.m3u8
			`,
		},
		{
			name: "advanced: absolute URL",
			args: args{
				input: `#EXTM3U
					#EXT-X-VERSION:3
					#EXT-X-KEY:METHOD=AES-128,URI="http://example.com/check",IV=0x00000000000000000000000000000000
					#EXTINF:2,
					http://example.com/01.ts
					#EXT-X-KEY:METHOD=AES-128,URI="http://example.com/check",IV=0x00000000000000000000000000000000
					#EXTINF:2,
					http://example.com/02.ts
					#EXT-X-KEY:METHOD=AES-128,URI="http://example.com/check",IV=0x00000000000000000000000000000000
					#EXTINF:2,
					http://example.com/03.ts
					#EXT-X-KEY:METHOD=AES-128,URI="http://example.com/check",IV=0x00000000000000000000000000000000
					#EXTINF:2,
					http://example.com/04.ts
				`,
				replace: func(s string) string { return strings.TrimPrefix(s, "http://example.com") },
			},
			want: `#EXTM3U
				#EXT-X-VERSION:3
				#EXT-X-KEY:METHOD=AES-128,URI="/check",IV=0x00000000000000000000000000000000
				#EXTINF:2,
				/01.ts
				#EXT-X-KEY:METHOD=AES-128,URI="/check",IV=0x00000000000000000000000000000000
				#EXTINF:2,
				/02.ts
				#EXT-X-KEY:METHOD=AES-128,URI="/check",IV=0x00000000000000000000000000000000
				#EXTINF:2,
				/03.ts
				#EXT-X-KEY:METHOD=AES-128,URI="/check",IV=0x00000000000000000000000000000000
				#EXTINF:2,
				/04.ts
			`,
		},
		{
			name: "advanced: realative URL",
			args: args{
				input: `#EXTM3U
					#EXT-X-VERSION:3
					#EXT-X-KEY:METHOD=AES-128,URI="/check",IV=0x00000000000000000000000000000000
					#EXTINF:2,
					/01.ts
					#EXT-X-KEY:METHOD=AES-128,URI="/check",IV=0x00000000000000000000000000000000
					#EXTINF:2,
					/02.ts
					#EXT-X-KEY:METHOD=AES-128,URI="/check",IV=0x00000000000000000000000000000000
					#EXTINF:2,
					/03.ts
					#EXT-X-KEY:METHOD=AES-128,URI="/check
					#EXTINF:2,
					/04.ts
				`,
				replace: func(s string) string { return "foo" + s },
			},
			want: `#EXTM3U
				#EXT-X-VERSION:3
				#EXT-X-KEY:METHOD=AES-128,URI="foo/check",IV=0x00000000000000000000000000000000
				#EXTINF:2,
				foo/01.ts
				#EXT-X-KEY:METHOD=AES-128,URI="foo/check",IV=0x00000000000000000000000000000000
				#EXTINF:2,
				foo/02.ts
				#EXT-X-KEY:METHOD=AES-128,URI="foo/check",IV=0x00000000000000000000000000000000
				#EXTINF:2,
				foo/03.ts
				#EXT-X-KEY:METHOD=AES-128,URI="/check
				#EXTINF:2,
				foo/04.ts
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want = strings.TrimSpace(tt.want)
			output := PlaylistUrlWalk(io.NopCloser(bytes.NewBuffer([]byte(tt.args.input))), tt.args.replace)

			// regexp remove whitespaces from start of all lines
			got := []byte(regexp.MustCompile(`(?m)^\s+`).ReplaceAll([]byte(output), []byte("")))
			got = bytes.TrimSpace(got)
			want := regexp.MustCompile(`(?m)^\s+`).ReplaceAll([]byte(tt.want), []byte(""))
			want = bytes.TrimSpace(want)

			if !reflect.DeepEqual(got, want) {
				t.Errorf("HlsRelativePathManifest() = \n---------- have ----------\n%s\n---------- want ----------\n%s", got, want)
			}
		})
	}
}

func TestRelativePath(t *testing.T) {
	type args struct {
		baseUrl string
		prefix  string
		u       string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "absolute URL",
			args: args{
				baseUrl: "http://example.com",
				prefix:  "/foo",
				u:       "http://example.com/bar",
			},
			want: "/foo/bar",
		},
		{
			name: "relative URL - start with /",
			args: args{
				baseUrl: "http://example.com",
				prefix:  "/test",
				u:       "/foo/bar",
			},
			want: "/test/foo/bar",
		},
		{
			name: "relative URL",
			args: args{
				baseUrl: "http://example.com",
				prefix:  "/foo",
				u:       "foo/bar",
			},
			want: "foo/bar",
		},
		{
			name: "relative URL - contains .",
			args: args{
				baseUrl: "http://example.com",
				prefix:  "/foo",
				u:       "foo/bar/./baz",
			},
			want: "foo/bar/baz",
		},
		{
			name: "relative URL - contains ..",
			args: args{
				baseUrl: "http://example.com",
				prefix:  "/foo",
				u:       "foo/bar/../baz",
			},
			want: "foo/baz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RelativePath(tt.args.baseUrl, tt.args.prefix, tt.args.u); got != tt.want {
				t.Errorf("RelativePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
