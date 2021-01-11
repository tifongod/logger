package logger

import (
	"context"
	"log"
	"net/http"
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

//easyjson:json
type Message struct {
	ServiceName string                 `json:"service_name"`
	Time        string                 `json:"date"`
	MessageType string                 `json:"message_type"`
	Trace       string                 `json:"trace,omitempty"`
	Source      string                 `json:"source,omitempty"`
	Stacktrace  *Stacktrace            `json:"stacktrace,omitempty"`
	Data        interface{}            `json:"data"`
	Tags        map[string]string      `json:"tags,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
	User        *UserForLog            `json:"user,omitempty"`
	Request     *http.Request          `json:"-"`
	Ctx         context.Context        `json:"-"`
}

//easyjson:json
type UserForLog struct {
	Email     string `json:"email,omitempty"`
	ID        string `json:"id,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
	Username  string `json:"username,omitempty"`
}

type blankMsg struct {
	level      int
	data       interface{}
	stack      []byte
	Stacktrace *Stacktrace
	ctx        context.Context

	mutator messageMutator
}

var levelSlug = map[int]string{
	ALERT: "ALERT",
	ERROR: "ERROR",
	LOG:   "LOG",
	DEBUG: "DEBUG",
	TRACE: "TRACE",
}

// GetLogger получение инстанса логгера
func GetLogger(config LoggerConfig) (*Logger, error) {
	l := &Logger{}
	for _, ld := range config.Output {
		err := ld.Init()
		if err != nil {
			return nil, err
		}
	}
	l.Config = config

	if l.Config.NeedToLog == nil {
		l.Config.NeedToLog = defaultNeedToLogDeterminant
	}

	in := make(chan blankMsg, config.Buffer)
	l.Msg = in
	go l.logging(l.Msg)

	return l, nil
}

func (l *Logger) logging(in chan blankMsg) {
	for msg := range in {
		for _, driver := range l.Config.Output {
			m := l.genMessage(msg.ctx, msg.level, msg.stack, msg.Stacktrace, msg.data)

			if msg.mutator != nil {
				m = msg.mutator.mutate(m)
			}

			err := driver.PutMsg(m.Msg)

			if err != nil {
				log.Fatalln(err)
			}
		}
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

func (l *Logger) genMessage(ctx context.Context, level int, stack []byte, stacktrace *Stacktrace, data interface{}) messages {
	code, ok := levelSlug[level]

	if !ok {
		code = "UNKNOWN"
	}

	var userForLog *UserForLog
	userForLogValue := ctx.Value("userForLog")
	if userForLogValue != nil {
		user, ok := userForLogValue.(*UserForLog)
		if ok {
			userForLog = user
		}
	}

	trace := string(stack)

	if err, ok := data.(error); ok {
		data = err.Error()
	}

	tags := l.extractTagsFromCtx(ctx)

	msg := messages{
		Level: level,
		Msg: Message{
			ServiceName: l.Config.ServiceName,
			Time:        time.Now().UTC().Format("2006-01-02 15:04:05"),
			MessageType: code,
			Data:        data,
			Tags:        tags,
			Trace:       trace,
			Stacktrace:  stacktrace,
			Ctx:         ctx,
			User:        userForLog,
		},
	}

	return msg
}

func (l *Logger) extractTagsFromCtx(ctx context.Context) map[string]string {
	res := make(map[string]string, len(l.Config.TagsFromCtx))
	for key, def := range l.Config.TagsFromCtx {
		resValue := def
		tmpVal := ctx.Value(key)
		if tmpVal != nil {
			valString, ok := tmpVal.(string)
			if ok {
				resValue = valString
			}
		}

		res[string(key)] = resValue
	}

	return res
}

type messageMutator interface {
	mutate(messages) messages
}

type LogEvent struct {
	l       *Logger
	Source  string                 `json:"source,omitempty"`
	Tags    map[string]string      `json:"tags,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
	User    *UserForLog            `json:"user,omitempty"`
	Request *http.Request          `json:"-"`
}

func (e *LogEvent) mutate(m messages) messages {
	if m.Msg.Tags == nil {
		m.Msg.Tags = e.Tags
	} else {
		for k, v := range e.Tags {
			m.Msg.Tags[k] = v
		}
	}

	m.Msg.Extra = e.Extra
	m.Msg.Request = e.Request

	if e.User != nil {
		m.Msg.User = e.User
	}

	return m
}

func (e *LogEvent) GetTags() map[string]string {
	return e.Tags
}

func (e *LogEvent) GetExtra() map[string]interface{} {
	return e.Extra
}

func (e *LogEvent) GetUser() *UserForLog {
	return e.User
}

func (e *LogEvent) GetRequest() *http.Request {
	return e.Request
}

func (e *LogEvent) WithTags(tags map[string]string) *LogEvent {
	e.Tags = tags
	return e
}

func (e *LogEvent) WithTag(k, v string) *LogEvent {
	if e.Tags == nil {
		e.Tags = make(map[string]string)
	}

	e.Tags[k] = v
	return e
}

func (e *LogEvent) WithExtras(extras map[string]interface{}) *LogEvent {
	e.Extra = extras
	return e
}

func (e *LogEvent) WithExtra(k string, v interface{}) *LogEvent {
	if e.Extra == nil {
		e.Extra = make(map[string]interface{})
	}

	e.Extra[k] = v
	return e
}

func (e *LogEvent) WithRequest(r *http.Request) *LogEvent {
	e.Request = r
	return e
}

func (e *LogEvent) WithUser(user *UserForLog) *LogEvent {
	e.User = user
	return e
}

func (e *LogEvent) log(ctx context.Context, level int, data interface{}) {
	if e.l.Config.NeedToLog(ctx, e.l.Config.Level, level) {
		stack := debug.Stack()
		bm := blankMsg{
			level:      level,
			data:       data,
			stack:      stack,
			Stacktrace: NewStacktrace(),
			ctx:        ctx,
			mutator:    e,
		}
		e.l.logMessage(bm)
	}
}

func (l *Logger) NewLogEvent() *LogEvent {
	return &LogEvent{l: l}
}

func (l *LogEvent) Alert(ctx context.Context, data interface{}) {
	l.log(ctx, ALERT, data)
}

func (l *LogEvent) Error(ctx context.Context, data interface{}) {
	l.log(ctx, ERROR, data)
}

func (l *LogEvent) Err(ctx context.Context, err error, msg string) {
	er := ErrorMsg{err: err, errText: msg}
	l.log(ctx, ERROR, er)
}

func (l *LogEvent) Log(ctx context.Context, data interface{}) {
	l.log(ctx, LOG, data)
}

func (l *LogEvent) Debug(ctx context.Context, data interface{}) {
	l.log(ctx, DEBUG, data)
}

func (l *LogEvent) Trace(ctx context.Context, data interface{}) {
	l.log(ctx, TRACE, data)
}
