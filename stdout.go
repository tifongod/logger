package logger

import (
	"fmt"
)

type STDOUTDriver struct {
}

func (s *STDOUTDriver) PutMsg(msg []byte) error {
	fmt.Println(string(msg))

	return nil
}
