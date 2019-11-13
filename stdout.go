package logger

import (
	"encoding/json"
	"log"
	"os"
)

type STDOUTDriver struct {
	baseLog *log.Logger
}

func (s *STDOUTDriver) Init() error {
	s.baseLog = log.New(os.Stdout, "", 0)
	return nil
}

func (s *STDOUTDriver) PutMsg(msg Message) error {
	logMsg, err := json.Marshal(msg)
	if err != nil {
		s.baseLog.Fatalln(err)
	}
	s.baseLog.Println(string(logMsg))

	return nil
}
