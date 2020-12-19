package api

import (
    "io"
	"fmt"
	"regexp"
    "os"
    "os/exec"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"
)

const (
	BUF_LEN = 1024
)

var conf *YamlConf
func init() {
	var err error
	conf, err = loadConf("/app/streams.yaml")
	if err != nil {
		log.Panic().Err(err).Msg("could not load `streams.yaml` file")
	}
}

type ApiManagerCtx struct {}

func New() *ApiManagerCtx {

	return &ApiManagerCtx{}
}

func (a *ApiManagerCtx) Mount(r *chi.Mux) {
	r.Get("/ping", func (w http.ResponseWriter, r *http.Request) {
		//nolint
		w.Write([]byte("pong"))
	})

	r.Get("/test", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")
		logger := log.With().
			Str("path", r.URL.Path).
			Str("module", "ffmpeg").
			Logger()
	
		logger.Info().Msg("command startred")
		cmd := exec.Command("/app/test.sh")
	
		read, write := io.Pipe() 
		cmd.Stdout = write
		cmd.Stderr = NewLogWriter(logger)

		defer func() {
			logger.Info().Msg("command stopped")

			read.Close()
			write.Close()
		}()

		go cmd.Run()
		io.Copy(w, read)
	})

	r.Get("/cpu/{input}/{profile}", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")
		logger := log.With().
			Str("path", r.URL.Path).
			Str("module", "ffmpeg").
			Logger()
	
		input := chi.URLParam(r, "input")
		url, ok := conf.Streams[input]
		if !ok {
			w.Write([]byte("stream not found"))
			logger.Warn().Msg("stream not found")
			return
		}

		profile := chi.URLParam(r, "profile")
		cmd, err := transcodeStart("profiles", profile, url)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("%v", err)))
			logger.Warn().Err(err).Msg("command failed")
			return
		}

		logger.Info().Msg("command startred")

		read, write := io.Pipe() 
		cmd.Stdout = write
		cmd.Stderr = NewLogWriter(logger)

		defer func() {
			logger.Info().Msg("command stopped")

			read.Close()
			write.Close()
		}()

		go cmd.Run()
		io.Copy(w, read)
	})

	r.Get("/gpu/{input}/{profile}", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")
		logger := log.With().
			Str("path", r.URL.Path).
			Str("module", "ffmpeg").
			Logger()

		input := chi.URLParam(r, "input")
		url, ok := conf.Streams[input]
		if !ok {
			w.Write([]byte("stream not found"))
			logger.Warn().Msg("stream not found")
			return
		}
	
		profile := chi.URLParam(r, "profile")
		cmd, err := transcodeStart("profiles_nvidia", profile, url)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("%v", err)))
			logger.Warn().Err(err).Msg("command failed")
			return
		}

		logger.Info().Msg("command startred")
	
		read, write := io.Pipe()
		cmd.Stdout = write
		cmd.Stderr = NewLogWriter(logger)

		go writeCmdOutput(w, read)
		cmd.Run()
		write.Close()
		logger.Info().Msg("command stopped")
	})
}

func writeCmdOutput(w http.ResponseWriter, read *io.PipeReader) {
	buffer := make([]byte, BUF_LEN)

	for {
		n, err := read.Read(buffer)
		if err != nil {
			read.Close()
			break
		}

		data := buffer[0:n]
		_, err = w.Write(data)
		if err != nil {
			read.Close()
			break
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		} else {
			log.Info().Msg("Damn, no flush")
		 }

		// reset buffer
		for i := 0; i < n; i++ {
			buffer[i] = 0
		}
	}
}

func transcodeStart(folder string, profile string, input string) (*exec.Cmd, error) {
	re := regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
	if !re.MatchString(folder) || !re.MatchString(profile) {
		return nil, fmt.Errorf("Invalid profile path.")
	}

	profilePath := fmt.Sprintf("/app/%s/%s.sh", folder, profile)
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return nil, err
	}

	log.Info().Str("profilePath", profilePath).Str("input", input).Msg("command startred")
	return exec.Command(profilePath, input), nil
}
