package util

import (
	"io"
	"os"
)

type StdCapture struct {
	savedStdout    *os.File
	reader, writer *os.File
}

func (s StdCapture) Cleanup() {
	os.Stdout = s.savedStdout
}

func (s StdCapture) Stop() {
	if err := s.writer.Close(); err != nil {
		panic(err)
	}
}

func (s StdCapture) Stdout() []byte {
	out, err := io.ReadAll(s.reader)
	if err != nil {
		panic(err)
	}
	return out
}

func NewStdCapture() *StdCapture {
	capture := &StdCapture{}
	capture.savedStdout = os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = writer
	capture.reader = reader
	capture.writer = writer

	return capture
}
