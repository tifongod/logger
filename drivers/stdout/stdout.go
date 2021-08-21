package stdout

import (
	"bytes"
	"fmt"
	"github.com/d-kolpakov/logger/v2"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"reflect"
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

//easyjson:json
type stdoutMsg struct {
	logger.Message
	FormattedStackTrace string `json:"fstacktrace,omitempty"`
	Request             string `json:"request,omitempty"`
}

func (s *STDOUTDriver) PutMsg(msg logger.Message) error {
	fmsg := stdoutMsg{Message: msg}

	needLogRequest := true
	needLogTrace := true
	fmsg.Data = parseData(fmsg.Data)

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

	logMsg, err := fmsg.MarshalJSON()
	if err != nil {
		s.baseLog.Fatalln(err)
	}
	s.baseLog.Println(string(logMsg))

	return nil
}

func (s *STDOUTDriver) formRequest(r *http.Request) string {
	res := ""

	if r.Body == nil {
		return res
	}

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

type dataMsg struct {
	DataMsg interface{} `json:"data_msg"`
}

func parseData (data interface{}) interface{} {
	ct := reflect.TypeOf(data)
	kind := ct.Kind()
	switch kind {
	case reflect.Struct, reflect.Map:
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		data = dataMsg{DataMsg: data}
	default:
		switch data.(type) {
		case []byte:
			if b, ok := data.([]byte); ok {
				data = string(b)
			}
		default:
			data = "Unknown object for log: " + kind.String()
		}
	}

	return data
}
