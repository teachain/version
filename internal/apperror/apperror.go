package apperror

import (
	"fmt"
)

type Kind int

const (
	NotFound Kind = iota + 1
	Conflict
	BadRequest
	Internal
)

func (k Kind) String() string {
	switch k {
	case NotFound:
		return "not found"
	case Conflict:
		return "conflict"
	case BadRequest:
		return "bad request"
	case Internal:
		return "internal"
	default:
		return "unknown"
	}
}

type Error struct {
	Kind  Kind
	Msg   string
	Cause error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %s", e.Kind, e.Msg, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Msg)
}

func (e *Error) Unwrap() error { return e.Cause }

func NotFoundf(format string, args ...any) *Error {
	return &Error{Kind: NotFound, Msg: fmt.Sprintf(format, args...)}
}

func Conflictf(format string, args ...any) *Error {
	return &Error{Kind: Conflict, Msg: fmt.Sprintf(format, args...)}
}

func BadRequestf(format string, args ...any) *Error {
	return &Error{Kind: BadRequest, Msg: fmt.Sprintf(format, args...)}
}

func InternalWrap(cause error, format string, args ...any) *Error {
	return &Error{Kind: Internal, Msg: fmt.Sprintf(format, args...), Cause: cause}
}