package logger

const (
	ALERT = iota
	ERROR
	LOG
	DEBUG
	TRACE
)

type LoggerConfig struct {
	Level  int
	Buffer int
	Output []LogDriver
}

type LogDriver interface {
	PutMsg(msg Message) error
}
