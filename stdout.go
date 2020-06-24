package logger

import (
	"encoding/json"
	"fmt"
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

type stdoutMsg struct {
	Message
	FormattedStackTrace string `json:"fstacktrace"`
}

func (s *STDOUTDriver) PutMsg(msg Message) error {
	fmsg := stdoutMsg{Message: msg}

	// Переформатируем вывод, т.к. елк не может нормально индексить и отображать слайсы
	if msg.Stacktrace != nil && msg.Stacktrace.Frames != nil {
		ftrace := ""
		for _, f := range msg.Stacktrace.Frames {
			ftrace += fmt.Sprintf(`%s in %s::%s at line %d\n`, f.AbsPath, f.Module, f.Function, f.Lineno)
		}
		fmsg.FormattedStackTrace = ftrace
		fmsg.Stacktrace = nil
	}

	logMsg, err := json.Marshal(fmsg)
	if err != nil {
		s.baseLog.Fatalln(err)
	}
	s.baseLog.Println(string(logMsg))

	return nil
}
