# hlsvod

When requesting this URL:

```
http://127.0.0.1:8080/hlsproxy/my_server1/index.m3u8
```

Following URL is requested from `go-transcode`:

```
http://192.168.1.34:9981/index.m3u8
```

Relative/absolute segment URLs in manifest are rewritten to relative and proxied too. If manifest segments are stored on external server, it may cause problems.
