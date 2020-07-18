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

type Message struct {
	ServiceName string                 `json:"service_name"`
	Time        string                 `json:"date"`
	RequestId   string                 `json:"request_id"`
	ClientID    string                 `json:"client_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	AccountID   string                 `json:"account_id,omitempty"`
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

type RequestUIDKey string
type ContextUIDKey string

var levelSlug = map[int]string{
	ALERT: "ALERT",
	ERROR: "ERROR",
	LOG:   "LOG",
	DEBUG: "DEBUG",
	TRACE: "TRACE",
}

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

func (l *Logger) log(ctx context.Context, level int, data interface{}) {
	if l.Config.Level >= level {
		stack := debug.Stack()
		bm := blankMsg{
			level:      level,
			data:       data,
			stack:      stack,
			Stacktrace: NewStacktrace(),
			ctx:        ctx,
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

func (l *Logger) genMessage(ctx context.Context, level int, stack []byte, stacktrace *Stacktrace, data interface{}) messages {
	code, ok := levelSlug[level]

	if !ok {
		code = "UNKNOWN"
	}

	requestId, clientID, accountID, userID := "no_context", "no_context", "", ""
	var userForLog *UserForLog

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
	var userIDKey ContextUIDKey = "userID"
	var userForLogKey ContextUIDKey = "userForLog"
	var sourceKey ContextUIDKey = "source"

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

	userForLogValue := ctx.Value(userForLogKey)
	if userForLogValue != nil {
		user, ok := userForLogValue.(*UserForLog)
		if ok {
			userForLog = user
		}
	}

	userIDValue := ctx.Value(userIDKey)
	if userIDValue != nil {
		idString, ok := userIDValue.(string)
		if ok {
			userID = idString
		}
	}

	source := ""
	sourceValue := ctx.Value(sourceKey)
	if sourceValue != nil {
		sourceString, ok := sourceValue.(string)
		if ok {
			source = sourceString
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
			Source:      source,
			AccountID:   accountID,
			Data:        data,
			Trace:       trace,
			Stacktrace:  stacktrace,
			Ctx:         ctx,
			User:        userForLog,
		},
	}

	return msg
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
	m.Msg.Source = e.Source
	m.Msg.Tags = e.Tags
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

func (e *LogEvent) WithSource(source string) *LogEvent {
	e.Source = source
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
	if e.l.Config.Level >= level {
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

func (l *LogEvent) AlertWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, ALERT, data)
}

func (l *LogEvent) ErrorWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, ERROR, data)
}

func (l *LogEvent) ErrWithContext(ctx context.Context, err error, msg string) {
	er := ErrorMsg{err: err, errText: msg}
	l.log(ctx, ERROR, er)
}

func (l *LogEvent) LogWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, LOG, data)
}

func (l *LogEvent) DebugWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, DEBUG, data)
}

func (l *LogEvent) TraceWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, TRACE, data)
}

func (l *LogEvent) Alert(data interface{}) {
	l.log(context.Background(), ALERT, data)
}

func (l *LogEvent) Err(data interface{}) {
	l.log(context.Background(), ERROR, data)
}

func (l *LogEvent) Error(msg string, err error) {
	er := ErrorMsg{err: err, errText: msg}
	l.log(context.Background(), ERROR, er)
}

func (l *LogEvent) Log(data interface{}) {
	l.log(context.Background(), LOG, data)
}

func (l *LogEvent) Debug(data interface{}) {
	l.log(context.Background(), DEBUG, data)
}

func (l *LogEvent) Trace(ctx context.Context, data interface{}) {
	l.log(context.Background(), TRACE, data)
}

// AlertWithContext deprecated
func (l *Logger) AlertWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, ALERT, data)
}

// ErrorWithContext deprecated
func (l *Logger) ErrorWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, ERROR, data)
}

// ErrWithContext deprecated
func (l *Logger) ErrWithContext(ctx context.Context, err error, msg string) {
	er := ErrorMsg{err: err, errText: msg}
	l.log(ctx, ERROR, er)
}

// LogWithContext deprecated
func (l *Logger) LogWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, LOG, data)
}

// DebugWithContext deprecated
func (l *Logger) DebugWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, DEBUG, data)
}

// TraceWithContext deprecated
func (l *Logger) TraceWithContext(ctx context.Context, data interface{}) {
	l.log(ctx, TRACE, data)
}

// Alert deprecated
func (l *Logger) Alert(data interface{}) {
	l.log(context.Background(), ALERT, data)
}

// Err deprecated
func (l *Logger) Err(data interface{}) {
	l.log(context.Background(), ERROR, data)
}

// Error deprecated
func (l *Logger) Error(msg string, err error) {
	er := ErrorMsg{err: err, errText: msg}
	l.log(context.Background(), ERROR, er)
}

// Log deprecated
func (l *Logger) Log(data interface{}) {
	l.log(context.Background(), LOG, data)
}

// Debug deprecated
func (l *Logger) Debug(data interface{}) {
	l.log(context.Background(), DEBUG, data)
}

// Trace deprecated
func (l *Logger) Trace(ctx context.Context, data interface{}) {
	l.log(context.Background(), TRACE, data)
}
