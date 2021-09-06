package utils

import (
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

const BUF_LEN = 1024

func IOPipeToHTTP(w http.ResponseWriter, read *io.PipeReader) {
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
			log.Info().Msg("damn, no flush")
		}

		// reset buffer
		for i := 0; i < n; i++ {
			buffer[i] = 0
		}
	}
}
