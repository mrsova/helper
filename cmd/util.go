package cmd

import (
	"io"
	"time"
)

func timeNow() string {
	return time.Now().Format("2006-01-02 15:04:05 MST")
}

func ioReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
