package errors

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// fixture functions doing work to avoid inlining
func a() error {
	if b(5) {
	    return nil
    }
    return fmt.Errorf("not gonna happen")
}

func b(i int) bool {
    return c(i + 2) > 12
}

func c(i int) int {
    if i > 3 {
	    panic('a')
    }
    return i * i
}

func TestParseStack(t *testing.T) {
	defer func() {
		err := New(recover(), 0)
        if err.Error() != "97" {
            t.Errorf("Received incorrect error, expected 'a' got '%s'", err.Error())
        }
		if err.TypeName() != "*errors.errorString" {
			t.Errorf("Error type was '%s'", err.TypeName())
		}
		expected := []StackFrame{
			StackFrame{Name: "TestParseStack.func1", File: "errors/error_test.go"},
			StackFrame{Name: "c", File: "errors/error_test.go", LineNumber: 25},
			StackFrame{Name: "b", File: "errors/error_test.go", LineNumber: 20},
			StackFrame{Name: "a", File: "errors/error_test.go", LineNumber: 13},
		}
        assertStacksMatch(t, expected, err.StackFrames())
	}()

	a()
}

func TestSkipWorks(t *testing.T) {
	defer func() {
		err := New(recover(), 1)
        if err.Error() != "97" {
            t.Errorf("Received incorrect error, expected 'a' got '%s'", err.Error())
        }

		expected := []StackFrame{
			StackFrame{Name: "c", File: "errors/error_test.go", LineNumber: 25},
			StackFrame{Name: "b", File: "errors/error_test.go", LineNumber: 20},
			StackFrame{Name: "a", File: "errors/error_test.go", LineNumber: 13},
		}

        assertStacksMatch(t, expected, err.StackFrames())
	}()

	a()
}

func checkFramesMatch(expected StackFrame, actual StackFrame) bool {
    if actual.Name != expected.Name {
        return false
    }
    // Not using exact match as it would change depending on whether
    // the package is being tested within or outside of the $GOPATH
    if expected.File != "" && !strings.HasSuffix(actual.File, expected.File) {
        return false
    }
    if expected.Package != "" && actual.Package != expected.Package {
        return false
    }
    if expected.LineNumber != 0 && actual.LineNumber != expected.LineNumber {
        return false
    }
    return true
}

func assertStacksMatch(t *testing.T, expected []StackFrame, actual []StackFrame) {
    var lastmatch int = 0
    for _, actualFrame := range actual {
        for index, expectedFrame := range expected {
            if index < lastmatch {
                continue
            }
            if checkFramesMatch(expectedFrame, actualFrame) {
                lastmatch = index
                break
            }
        }
    }
    if lastmatch != len(expected) - 1 {
        t.Fatalf("failed to find matches for %d frames: '%v'\ngot: '%v'", len(expected) - lastmatch, expected[lastmatch:], actual)
    }
}

type testErrorWithStackFrames struct {
	Err *Error
}

func (tews *testErrorWithStackFrames) StackFrames() []StackFrame {
	return tews.Err.StackFrames()
}

func (tews *testErrorWithStackFrames) Error() string {
	return tews.Err.Error()
}

func TestNewError(t *testing.T) {

	e := func() error {
		return New("hi", 1)
	}()

	if e.Error() != "hi" {
		t.Errorf("Constructor with a string failed")
	}

	if New(fmt.Errorf("yo"), 0).Error() != "yo" {
		t.Errorf("Constructor with an error failed")
	}

	if New(e, 0) != e {
		t.Errorf("Constructor with an Error failed")
	}

	if New(nil, 0).Error() != "<nil>" {
		t.Errorf("Constructor with nil failed")
	}

	err := New("foo", 0)
	tews := &testErrorWithStackFrames{
		Err: err,
	}

	if bytes.Compare(New(tews, 0).Stack(), err.Stack()) != 0 {
		t.Errorf("Constructor with ErrorWithStackFrames failed")
	}
}

func ExampleErrorf() {
	for i := 1; i <= 2; i++ {
		if i%2 == 1 {
			e := Errorf("can only halve even numbers, got %d", i)
			fmt.Printf("Error: %+v", e)
		}
	}
	// Output:
	// Error: can only halve even numbers, got 1
}

func ExampleNew() {
	// Wrap io.EOF with the current stack-trace and return it
	e := New(io.EOF, 0)
	fmt.Printf("%+v", e)
	// Output:
	// EOF
}

func ExampleNew_skip() {
	defer func() {
		if err := recover(); err != nil {
			// skip 1 frame (the deferred function) and then return the wrapped err
			err = New(err, 1)
		}
	}()
}
