package apperror

import (
	"errors"
	"fmt"
	"testing"
)

func TestError_ErrorMessage(t *testing.T) {
	e := &Error{Kind: NotFound, Msg: "user 42"}
	if got, want := e.Error(), "not found: user 42"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("boom")
	e := &Error{Kind: Internal, Msg: "wrap", Cause: cause}
	if !errors.Is(e, cause) {
		t.Fatalf("errors.Is should match wrapped cause")
	}
}

func TestConstructors(t *testing.T) {
	if got, want := NotFoundf("x %d", 1).Kind, NotFound; got != want {
		t.Errorf("got kind=%v, want=%v", got, want)
	}
	if got, want := Conflictf("x").Kind, Conflict; got != want {
		t.Errorf("got kind=%v, want=%v", got, want)
	}
	if got, want := BadRequestf("x").Kind, BadRequest; got != want {
		t.Errorf("got kind=%v, want=%v", got, want)
	}
	if got, want := InternalWrap(fmt.Errorf("e"), "x").Kind, Internal; got != want {
		t.Errorf("got kind=%v, want=%v", got, want)
	}
}

func TestError_ErrorMessageWithCause(t *testing.T) {
	cause := errors.New("boom")
	e := &Error{Kind: Internal, Msg: "wrap", Cause: cause}
	if got, want := e.Error(), "internal: wrap: boom"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}