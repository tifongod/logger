package logger

import "fmt"

type ErrorMsg struct {
	err     error
	errText string
}

func (e *ErrorMsg) Error() string {
	return fmt.Sprintf("error: %s, error_message: %s", e.err.Error(), e.errText)
}
