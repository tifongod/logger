package logger

import (
	"encoding/json"
	"log"
)

type STDOUTDriver struct {
}

func (s *STDOUTDriver) PutMsg(msg message) error {
	logMsg, err := json.Marshal(msg)

	if err != nil {
		log.Fatalln(err)
	}

	log.Println(string(logMsg))

	return nil
}
