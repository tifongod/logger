package logger

import (
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"reflect"
	"time"
)

const maxErrorDepth = 10

type SentryDriver struct {
	ClientOptions *sentry.ClientOptions
	Client        *sentry.Client
	FlushTimeout  time.Duration
	NeedToCapture map[string]sentry.Level
	IsErrorEvent  map[string]struct{}
}

var defaultCaptured = map[string]sentry.Level{
	"ALERT":   sentry.LevelFatal,
	"UNKNOWN": sentry.LevelError,
}

var defaultErrorEvents = map[string]struct{}{
	"ALERT":   struct{}{},
	"UNKNOWN": struct{}{},
}

func (s *SentryDriver) Init() error {
	if s.Client == nil && s.ClientOptions != nil {
		cl, err := sentry.NewClient(*s.ClientOptions)

		if err != nil {
			return err
		}
		s.Client = cl
	}
	if s.NeedToCapture == nil || len(s.NeedToCapture) <= 0 {
		s.NeedToCapture = defaultCaptured
	}

	if s.IsErrorEvent == nil || len(s.IsErrorEvent) <= 0 {
		s.IsErrorEvent = defaultErrorEvents
	}

	if s.FlushTimeout <= 0 {
		s.FlushTimeout = 2 * time.Second
	}

	return nil
}

func (s *SentryDriver) PutMsg(msg Message) error {
	level, ok := s.NeedToCapture[msg.MessageType]

	if !ok {
		return nil
	}

	defer s.Client.Flush(s.FlushTimeout)

	scope := sentry.NewScope()

	tags := msg.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	tags["source"] = msg.Source

	scope.SetTags(msg.Tags)
	scope.SetExtras(msg.Extra)

	if msg.Request != nil {
		scope.SetRequest(msg.Request)
	}

	if msg.User != nil {
		scope.SetUser(sentry.User{
			Email:     msg.User.Email,
			ID:        msg.User.ID,
			IPAddress: msg.User.IPAddress,
			Username:  msg.User.Username,
		})
	}

	var event *sentry.Event

	if _, ok := s.IsErrorEvent[msg.MessageType]; ok {
		event = eventFromException(msg, level)
	} else {
		event = eventFromMessage(msg, level)
	}

	s.Client.CaptureEvent(event, nil, scope)

	return nil
}

func eventFromException(msg Message, level sentry.Level) *sentry.Event {
	var err, capturedError error

	if err, ok := msg.Data.(ErrorMsg); ok {
		capturedError = err.err
	} else if err, ok := msg.Data.(error); ok {
		capturedError = err
	} else if err, ok := msg.Data.(string); ok {
		capturedError = errors.New(err)
	} else {
		capturedError = errors.New(fmt.Sprintf("unknown error: %v", msg.Data))
	}

	err = capturedError

	event := sentry.NewEvent()
	event.Level = level

	for i := 0; i < maxErrorDepth && err != nil; i++ {
		event.Exception = append(event.Exception, sentry.Exception{
			Value:      err.Error(),
			Type:       reflect.TypeOf(err).String(),
			Stacktrace: convertLoggerTraceToSentryTrace(ExtractStacktrace(err)),
		})
		switch previous := err.(type) {
		case interface{ Unwrap() error }:
			err = previous.Unwrap()
		case interface{ Cause() error }:
			err = previous.Cause()
		default:
			err = nil
		}
	}

	// Add a trace of the current stack to the most recent error in a chain if
	// it doesn't have a stack trace yet.
	// We only add to the most recent error to avoid duplication and because the
	// current stack is most likely unrelated to errors deeper in the chain.
	if event.Exception[0].Stacktrace == nil {
		event.Exception[0].Stacktrace = convertLoggerTraceToSentryTrace(msg.Stacktrace)
	}

	// event.Exception should be sorted such that the most recent error is last.
	reverse(event.Exception)

	return event
}

func convertLoggerTraceToSentryTrace(stacktrace *Stacktrace) *sentry.Stacktrace {
	res := &sentry.Stacktrace{}

	if stacktrace == nil {
		return nil
	}

	res.FramesOmitted = stacktrace.FramesOmitted

	if stacktrace.Frames != nil {
		resFrames := make([]sentry.Frame, 0, len(stacktrace.Frames))
		for _, f := range stacktrace.Frames {
			resFrames = append(resFrames, sentry.Frame{
				Function:    f.Function,
				Symbol:      f.Symbol,
				Module:      f.Module,
				Package:     f.Package,
				Filename:    f.Filename,
				AbsPath:     f.AbsPath,
				Lineno:      f.Lineno,
				Colno:       f.Colno,
				PreContext:  f.PreContext,
				ContextLine: f.ContextLine,
				PostContext: f.PostContext,
				InApp:       f.InApp,
				Vars:        f.Vars,
			})
		}

		res.Frames = resFrames
	}

	return res
}

func reverse(a []sentry.Exception) {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}

func eventFromMessage(msg Message, level sentry.Level) *sentry.Event {
	message := ""
	if data, ok := msg.Data.(string); ok {
		message = data
	} else {
		message = fmt.Sprintf("%v", msg.Data)
	}

	event := sentry.NewEvent()
	event.Level = level
	event.Message = message

	event.Threads = []sentry.Thread{{
		Stacktrace: convertLoggerTraceToSentryTrace(msg.Stacktrace),
		Crashed:    false,
		Current:    true,
	}}

	return event
}
