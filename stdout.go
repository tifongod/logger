package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

type STDOUTDriver struct {
	LogRequest map[string]struct{}
	LogTrace   map[string]struct{}
	baseLog    *log.Logger
}

func (s *STDOUTDriver) Init() error {
	s.baseLog = log.New(os.Stdout, "", 0)
	return nil
}

type stdoutMsg struct {
	Message
	FormattedStackTrace string `json:"fstacktrace,omitempty"`
	Request             string `json:"request,omitempty"`
}

func (s *STDOUTDriver) PutMsg(msg Message) error {
	fmsg := stdoutMsg{Message: msg}

	needLogRequest := true
	needLogTrace := true

	if s.LogRequest != nil && len(s.LogRequest) > 0 {
		_, needLogRequest = s.LogRequest[msg.MessageType]
	}

	if s.LogTrace != nil && len(s.LogTrace) > 0 {
		_, needLogTrace = s.LogTrace[msg.MessageType]
	}

	// Переформатируем вывод, т.к. елк не может нормально индексить и отображать слайсы
	if needLogTrace && msg.Stacktrace != nil && msg.Stacktrace.Frames != nil {
		ftrace := ""
		for _, f := range msg.Stacktrace.Frames {
			ftrace += fmt.Sprintf(`%s in %s::%s at line %d\n`, f.AbsPath, f.Module, f.Function, f.Lineno)
		}
		fmsg.FormattedStackTrace = ftrace
		fmsg.Stacktrace = nil
	}

	if needLogRequest && msg.Request != nil {
		fmsg.Request = s.formRequest(msg.Request)
	}

	logMsg, err := json.Marshal(fmsg)
	if err != nil {
		s.baseLog.Fatalln(err)
	}
	s.baseLog.Println(string(logMsg))

	return nil
}

func (s *STDOUTDriver) formRequest(r *http.Request) string {
	res := ""

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.baseLog.Println(err.Error())
		return res
	}

	r.Body.Close()
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	byteDump, err := httputil.DumpRequest(r, false)

	if err != nil {
		s.baseLog.Println(err.Error())
		return res
	}

	res = string(byteDump)

	return res
}
