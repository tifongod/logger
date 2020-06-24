package logger

import (
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"time"
)

type SentryDriver struct {
	ClientOptions *sentry.ClientOptions
	Client        *sentry.Client
	FlushTimeout  time.Duration
	NeedToCapture map[string]struct{}
}

func (s *SentryDriver) Init() error {
	if s.Client == nil && s.ClientOptions != nil {
		cl, err := sentry.NewClient(*s.ClientOptions)

		if err != nil {
			return err
		}
		s.Client = cl
	}

	defaultCaptured := map[string]struct{}{
		"ALERT":   {},
		"UNKNOWN": {},
	}

	if s.NeedToCapture == nil || len(s.NeedToCapture) <= 0 {
		s.NeedToCapture = defaultCaptured
	}

	if s.FlushTimeout <= 0 {
		s.FlushTimeout = 2 * time.Second
	}

	return nil
}

func (s *SentryDriver) PutMsg(msg Message) error {
	_, ok := s.NeedToCapture[msg.MessageType]

	if !ok {
		return nil
	}

	defer s.Client.Flush(s.FlushTimeout)

	scope := sentry.NewScope()

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

	var capturedError error

	if err, ok := msg.Data.(ErrorMsg); ok {
		capturedError = err.err
		scope.SetExtra("errText", err.errText)
	} else if err, ok := msg.Data.(error); ok {
		capturedError = err
	} else if err, ok := msg.Data.(string); ok {
		capturedError = errors.New(err)
	} else {
		capturedError = errors.New(fmt.Sprintf("unknown error: %v", msg.Data))
	}

	s.Client.CaptureException(capturedError, nil, scope)

	return nil
}
