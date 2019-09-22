package logger

import (
	"io"
)

type STDOUTDriver struct {
	Writer io.Writer
}

func (s *STDOUTDriver)PutMsg(msg []byte) error {
	_, err := s.Writer.Write(msg)

	return err
}
