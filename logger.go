package logger

import (
	"context"
	"log"
	"runtime/debug"
	"strings"
	"time"
)

type Logger struct {
	Config LoggerConfig
	Msg    chan blankMsg
}

type messages struct {
	Level int
	Msg   Message
}

type Message struct {
	ServiceName string          `json:"service_name"`
	Time        string          `json:"time"`
	RequestId   string          `json:"request_id"`
	MessageType string          `json:"message_type"`
	Trace       []string        `json:"trace"`
	Data        interface{}     `json:"data"`
	Ctx         context.Context `json:"-"`
}

type blankMsg struct {
	level int
	data  interface{}
	stack []byte
	ctx   context.Context
}

type RequestUIDKey string

// GetLogger получение инстанса логгер
func GetLogger(config LoggerConfig) (*Logger, error) {
	l := &Logger{}
	l.Config = config
	in := make(chan blankMsg, config.Buffer)
	l.Msg = in
	go l.logging(l.Msg)

	return l, nil
}

func (l *Logger) logging(in chan blankMsg) {
	for msg := range in {
		for _, driver := range l.Config.Output {
			m := l.genMessage(msg.ctx, msg.level, msg.stack, msg.data)
			err := driver.PutMsg(m.Msg)

			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func (l *Logger) log(ctx context.Context, level int, data interface{}) {
	if l.Config.Level >= level {
		stack := debug.Stack()
		bm := blankMsg{
			level: level,
			data:  data,
			stack: stack,
			ctx:   ctx,
		}
		l.logMessage(bm)
	}
}

func (l *Logger) logMessage(msg blankMsg) {
	select {
	case l.Msg <- msg:
	case <-time.After(time.Microsecond * 20):
		ch := make(chan blankMsg)
		go l.logging(ch)
		defer close(ch)
	}
}

func (l *Logger) genMessage(ctx context.Context, level int, stack []byte, data interface{}) messages {
	var code string
	switch level {
	case ALERT:
		code = "ALERT"
	case ERROR:
		code = "ERROR"
	case LOG:
		code = "LOG"
	case DEBUG:
		code = "DEBUG"
	case TRACE:
		code = "TRACE"
	}

	requestId := "no_context"

	var key RequestUIDKey = "requestID"
	id := ctx.Value(key)
	if id != nil {
		idString, ok := id.(string)
		if ok {
			requestId = idString
		}
	}
	trace := strings.Split(string(stack), "\n")

	if err, ok := data.(error); ok {
		data = err.Error()
	}

	msg := messages{
		Level: level,
		Msg: Message{
			ServiceName: l.Config.ServiceName,
			Time:        time.Now().UTC().Format("2006-01-02 15:04:05"),
			MessageType: code,
			RequestId:   requestId,
			Data:        data,
			Trace:       trace[7:],
			Ctx:         ctx,
		},
	}

	return msg
}

func (l *Logger) AlertWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, ALERT, data)
}

func (l *Logger) ErrorWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, ERROR, data)
}

func (l *Logger) ErrWithContext(ctx context.Context, err error, msg string) {
	er := ErrorMsg{err: err, errText: msg}
	l.log(ctx, ERROR, er)
}

func (l *Logger) LogWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, LOG, data)
}

func (l *Logger) DebugWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, DEBUG, data)
}

func (l *Logger) TraceWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, TRACE, data)
}

func (l *Logger) Alert(ctx context.Context, data interface{}) {
	l.log(context.Background(), ALERT, data)
}

func (l *Logger) Error(ctx context.Context, data interface{}) {
	l.log(context.Background(), ERROR, data)
}

func (l *Logger) Err(ctx context.Context, err error, msg string) {
	er := ErrorMsg{err: err, errText: msg}
	l.log(context.Background(), ERROR, er)
}

func (l *Logger) Log(ctx context.Context, data interface{}) {
	l.log(context.Background(), LOG, data)
}

func (l *Logger) Debug(ctx context.Context, data interface{}) {
	l.log(context.Background(), DEBUG, data)
}

func (l *Logger) Trace(ctx context.Context, data interface{}) {
	l.log(context.Background(), TRACE, data)
}
