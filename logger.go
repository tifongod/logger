package logger

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/satori/go.uuid"
)

type Logger struct {
	Config LoggerConfig
	Msg    chan messages
}

type messages struct {
	Level int
	Msg   message
}

type message struct {
	Time      string
	RequestId string
	Code      string
	Data      interface{}
	Ctx       context.Context
}

type RequestUIDKey string

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
		if l.Config.Level < msg.Level {
			continue
		}

		logMsg, err := json.Marshal(msg.Msg)

		if err != nil {
			log.Fatalln(err)
			continue
		}

		for _, driver := range l.Config.Output {

			err := driver.PutMsg(logMsg)

			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func (l *Logger) log(msg messages) {
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

	requestId := uuid.NewV4().String()

	var key RequestUIDKey = "requestId"
	id := ctx.Value(key)

	if id != nil {
		idString, ok := id.(string)

		if ok {
			requestId = idString
		}
	}

	msg := messages{
		Level: level,
		Msg: message{
			Time:      time.Now().UTC().Format("2006-01-02 15:04:05"),
			Code:      code,
			RequestId: requestId,
			Data:      data,
			Ctx:       ctx,
		},
	}

	return msg
}

func (l *Logger) Alert(ctx context.Context, data interface{}) {
	l.log(l.genMessage(ctx, ALERT, data))
}

func (l *Logger) Error(ctx context.Context, data interface{}) {
	l.log(l.genMessage(ctx, ERROR, data))
}

func (l *Logger) Log(ctx context.Context, data interface{}) {
	l.log(l.genMessage(ctx, LOG, data))
}

func (l *Logger) Debug(ctx context.Context, data interface{}) {
	l.log(l.genMessage(ctx, DEBUG, data))
}

func (l *Logger) Trace(ctx context.Context, data interface{}) {
	l.log(l.genMessage(ctx, TRACE, data))
}
