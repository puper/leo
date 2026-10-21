package engine

import (
	"errors"
	"testing"
)

type failingCloser struct {
	closeErr error
}

func (f *failingCloser) Close() error {
	return f.closeErr
}

func TestClosePropagatesCloserErrors(t *testing.T) {
	e := New(nil)

	closeErr := errors.New("close failed")
	e.Register("failing", func() (any, error) {
		return &failingCloser{closeErr: closeErr}, nil
	})

	if err := e.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	result := e.Close()
	if result == nil {
		t.Error("BUG FIXED: Close() should return error but got nil")
	} else if result.Error() != "close failed" {
		t.Errorf("Close() returned wrong error: got %v, want 'close failed'", result)
	} else {
		t.Logf("FIX VERIFIED: Close() correctly propagated error: %v", result)
	}
}

type mockBuilder struct {
	name    string
	buildFn func() (any, error)
}

func (me *mockBuilder) Build() (any, error) {
	return me.buildFn()
}

func TestBuildDuplicateTopoOrderingCalls(t *testing.T) {
	e := New(nil)

	buildCount := 0
	e.Register("test1", func() (any, error) {
		buildCount++
		return "instance1", nil
	})
	e.Register("test2", func() (any, error) {
		return "instance2", nil
	}, "test1")

	if err := e.Build(); err != nil {
		t.Fatal(err)
	}

	if buildCount != 1 {
		t.Fatalf("test1 built %d times, want 1", buildCount)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if buildCount != 1 {
		t.Fatalf("Close rebuilt test1: count = %d", buildCount)
	}
}

func TestEngineGetPanicsWhenNotFound(t *testing.T) {
	e := New(nil)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Get() did not panic for a missing component")
		}
	}()

	e.Get("nonexistent")
}
