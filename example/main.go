package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/d-kolpakov/logger"
	sentryDriver "github.com/d-kolpakov/logger/drivers/sentry"
	"github.com/d-kolpakov/logger/drivers/stdout"
	"github.com/getsentry/sentry-go"
	"time"
)

func main() {
	lDrivers := make([]logger.LogDriver, 0, 5)

	stdoutLD := &stdout.STDOUTDriver{}

	option := &sentry.ClientOptions{
		Dsn:              "",
		Debug:            true,
		AttachStacktrace: true,
		Environment:      "local",
	}

	sentryD := &sentryDriver.SentryDriver{
		ClientOptions: option,
		NeedToCapture: map[string]sentry.Level{
			"ALERT":   sentry.LevelFatal,
			"UNKNOWN": sentry.LevelError,
			"DEBUG":   sentry.LevelDebug,
		},
	}

	lDrivers = append(lDrivers, stdoutLD)
	lDrivers = append(lDrivers, sentryD)

	lc := logger.LoggerConfig{
		ServiceName: "test",
		Level:       logger.DEBUG,
		Buffer:      1000,
		Output:      lDrivers,
		TagsFromCtx: map[logger.ContextUIDKey]string{
			logger.ContextUIDKey("tag1"):      "empty",
			logger.ContextUIDKey("requestId"): "empty",
			logger.ContextUIDKey("source"):    "empty",
			logger.ContextUIDKey("accountId"): "empty",
		},
	}
	l, err := logger.GetLogger(lc)

	if err != nil {
		panic(err)
	}

	l.NewLogEvent().Debug(context.Background(), fmt.Sprintf(`start %s service`, "test"))

	ctx := context.Background()
	ctx = context.WithValue(ctx, logger.ContextUIDKey("requestId"), "4fd7d2c0-df29-4ad8-b6e3-1d0c2805a5bf")
	ctx = context.WithValue(ctx, logger.ContextUIDKey("source"), "example")
	ctx = context.WithValue(ctx, logger.ContextUIDKey("accountId"), "11728654")

	l.NewLogEvent().WithTag("is_done", "yeap").WithExtra("ddd", 54).Alert(ctx, errors.New("very new alert"))
	l.NewLogEvent().WithTag("is_new", "true").WithExtra("xxx", 5412).Alert(ctx, errors.New("хочу увидеть стек-трейс"))
	l.NewLogEvent().WithTag("is_new", "true").WithExtra("xxx", 5412).Debug(ctx, errors.New("хочу увидеть стек-трейс дебага"))
	l.NewLogEvent().WithTag("is_new", "true").WithExtra("xxx", 5412).Trace(ctx, errors.New("не выведет, так как Trace выше Debug"))

	time.Sleep(time.Second * 100)
}
