package logger

import (
	"context"
)

const (
	ALERT = iota
	ERROR
	LOG
	DEBUG
	TRACE
)

type LoggerConfig struct {
	ServiceName string
	Level       int
	Buffer      int
	Output      []LogDriver
	TagsFromCtx map[ContextUIDKey]string
	NeedToLog   NeedToLogDeterminant
}

type LogDriver interface {
	PutMsg(msg Message) error
	Init() error
}

type NeedToLogDeterminant func(ctx context.Context, configuredLevel, level int) bool

var defaultNeedToLogDeterminant = func(ctx context.Context, configuredLevel, level int) bool {
	if configuredLevel >= level {
		return true
	}

	return false
}
