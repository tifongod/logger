package logger

import (
	"context"
	"log"
	"runtime/debug"
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
	Time        string          `json:"date"`
	RequestId   string          `json:"request_id"`
	ClientID    string          `json:"client_id,omitempty"`
	UserID      string          `json:"user_id,omitempty"`
	AccountID   string          `json:"account_id,omitempty"`
	MessageType string          `json:"message_type"`
	Trace       string          `json:"trace"`
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
type ContextUIDKey string

// GetLogger получение инстанса логгер
func GetLogger(config LoggerConfig) (*Logger, error) {
	l := &Logger{}
	for _, ld := range config.Output {
		err := ld.Init()
		if err != nil {
			return nil, err
		}
	}
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

	requestId, clientID, accountID, userID := "no_context", "no_context", "", ""

	var key RequestUIDKey = "requestID"
	id := ctx.Value(key)
	if id != nil {
		idString, ok := id.(string)
		if ok {
			requestId = idString
		}
	}

	var clientIDKey ContextUIDKey = "clientID"
	var accountIDKey ContextUIDKey = "accountID"
	var userIDKey ContextUIDKey = "accountID"

	clientIDValue := ctx.Value(clientIDKey)
	if clientIDValue != nil {
		idString, ok := clientIDValue.(string)
		if ok {
			clientID = idString
		}
	}

	accountIDValue := ctx.Value(accountIDKey)
	if accountIDValue != nil {
		idString, ok := accountIDValue.(string)
		if ok {
			accountID = idString
		}
	}

	userIDValue := ctx.Value(userIDKey)
	if userIDValue != nil {
		idString, ok := userIDValue.(string)
		if ok {
			userID = idString
		}
	}

	trace := string(stack)

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
			ClientID:    clientID,
			UserID:      userID,
			AccountID:   accountID,
			Data:        data,
			Trace:       trace,
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

func (l *Logger) Alert(data interface{}) {
	l.log(context.Background(), ALERT, data)
}

func (l *Logger) Err(data interface{}) {
	l.log(context.Background(), ERROR, data)
}

func (l *Logger) Error(msg string, err error) {
	er := ErrorMsg{err: err, errText: msg}
	l.log(context.Background(), ERROR, er)
}

func (l *Logger) Log(data interface{}) {
	l.log(context.Background(), LOG, data)
}

func (l *Logger) Debug(data interface{}) {
	l.log(context.Background(), DEBUG, data)
}

func (l *Logger) Trace(ctx context.Context, data interface{}) {
	l.log(context.Background(), TRACE, data)
}
