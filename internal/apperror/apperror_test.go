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
	if NotFoundf("x %d", 1).Kind != NotFound {
		t.Fail()
	}
	if Conflictf("x").Kind != Conflict {
		t.Fail()
	}
	if BadRequestf("x").Kind != BadRequest {
		t.Fail()
	}
	if InternalWrap(fmt.Errorf("e"), "x").Kind != Internal {
		t.Fail()
	}
}