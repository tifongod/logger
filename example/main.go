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
		Level:       logger.TRACE,
		Buffer:      1000,
		Output:      lDrivers,
	}
	l, err := logger.GetLogger(lc)

	if err != nil {
		panic(err)
	}

	l.NewLogEvent().Debug(context.Background(), fmt.Sprintf(`start %s service`, "test"))

	l.NewLogEvent().WithTag("is_done", "yeap").WithExtra("ddd", 54).Alert(context.Background(), errors.New("very new alert"))
	l.NewLogEvent().WithTag("is_new", "true").WithExtra("xxx", 5412).Alert(context.Background(), errors.New("хочу увидеть стек-трейс"))
	l.NewLogEvent().WithTag("is_new", "true").WithExtra("xxx", 5412).Debug(context.Background(), errors.New("хочу увидеть стек-трейс дебага"))

	time.Sleep(time.Second * 100)
}
