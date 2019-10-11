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
	Msg    chan messages
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

type RequestUIDKey string

// GetLogger получение инстанса логгер
func GetLogger(config LoggerConfig) (*Logger, error) {
	l := &Logger{}

	l.Config = config
	in := make(chan messages, config.Buffer)
	l.Msg = in

	go l.logging(l.Msg)

	return l, nil
}

func (l *Logger) logging(in chan messages) {
	for msg := range in {
		for _, driver := range l.Config.Output {
			err := driver.PutMsg(msg.Msg)

			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func (l *Logger) log(ctx context.Context, level int, data interface{}) {
	if l.Config.Level >= level {
		l.logMessage(l.genMessage(ctx, level, data))
	}
}

func (l *Logger) logMessage(msg messages) {
	select {
	case l.Msg <- msg:
	case <-time.After(time.Microsecond * 20):
		ch := make(chan messages)
		go l.logging(ch)
		defer close(ch)
	}
}

func (l *Logger) genMessage(ctx context.Context, level int, data interface{}) messages {
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

	var key RequestUIDKey = "requestId"
	id := ctx.Value(key)

	if id != nil {
		idString, ok := id.(string)

		if ok {
			requestId = idString
		}
	}

	trace := string(debug.Stack())

	if err, ok := data.(error); ok {
		data = err.Error()
	}

	msg := messages{
		Level: level,
		Msg: Message{
			Time:        time.Now().UTC().Format("2006-01-02 15:04:05"),
			MessageType: code,
			RequestId:   requestId,
			Data:        data,
			Trace:       strings.Split(trace, "\n"),
			Ctx:         ctx,
		},
	}

	return msg
}

func (l *Logger) Alert(ctx context.Context, data interface{}) {
	l.log(ctx, ALERT, data)
}

func (l *Logger) Error(ctx context.Context, data interface{}) {
	l.log(ctx, ERROR, data)
}

func (l *Logger) Err(ctx context.Context, err error) {
	l.log(ctx, ERROR, err)
}

func (l *Logger) Log(ctx context.Context, data interface{}) {
	l.log(ctx, LOG, data)
}

func (l *Logger) Debug(ctx context.Context, data interface{}) {
	l.log(ctx, DEBUG, data)
}

func (l *Logger) Trace(ctx context.Context, data interface{}) {
	l.log(ctx, TRACE, data)
}
