package errors

import "fmt"

type BizError struct {
	Code       int
	MessageKey string
	Args       map[string]any
	Cause      error
}

func NewBizError(code int, messageKey string, args ...map[string]any) *BizError {
	err := &BizError{Code: code, MessageKey: messageKey}
	if len(args) > 0 {
		err.Args = cloneArgs(args[0])
	}
	return err
}

func (e *BizError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.MessageKey, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.MessageKey)
}

func (e *BizError) WithCause(err error) *BizError {
	e.Cause = err
	return e
}

func (e *BizError) Unwrap() error {
	return e.Cause
}

func cloneArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	out := make(map[string]any, len(args))
	for key, value := range args {
		out[key] = value
	}
	return out
}
