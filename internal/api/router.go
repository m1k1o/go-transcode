package api

import (
    "io"
    "os/exec"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"
)

const (
	BUF_LEN = 1024
)

type ApiManagerCtx struct {}

func New() *ApiManagerCtx {

	return &ApiManagerCtx{}
}

func (a *ApiManagerCtx) Mount(r *chi.Mux) {
	r.Get("/ping", func (w http.ResponseWriter, r *http.Request) {
		//nolint
		w.Write([]byte("pong"))
	})

	r.Get("/test1", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")
	
		log.Info().Msg("command Start")
		cmd := exec.Command("/app/test.sh")
	
		read, write := io.Pipe() 
		cmd.Stdout = write

		defer func() {
			log.Info().Msg("command Stop")

			read.Close()
			write.Close()
		}()

		go cmd.Run()
		io.Copy(w, read)
	})

	r.Get("/test2", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")

		log.Info().Msg("command Start")
		cmd := exec.Command("/app/test.sh")

		read, write := io.Pipe()
		cmd.Stdout = write

		go writeCmdOutput(w, read)
		cmd.Run()
		write.Close()
		log.Info().Msg("command Stop")
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
