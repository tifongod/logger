package logger

import (
	"encoding/json"
	"log"
)

type STDOUTDriver struct {
}

func (s *STDOUTDriver) PutMsg(msg Message) error {
	logMsg, err := json.Marshal(msg)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(string(logMsg))

	return nil
}
