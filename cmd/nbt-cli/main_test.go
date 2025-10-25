package main

import (
	"fmt"
	"testing"
)

func TestExitError(t *testing.T) {
	err := exitError(3, fmt.Errorf("boom"))
	ec, ok := err.(exitCoder)
	if !ok {
		t.Fatalf("error does not implement exitCoder: %T", err)
	}
	if got := ec.ExitCode(); got != 3 {
		t.Fatalf("exit code: got %d, want 3", got)
	}
	if got := err.Error(); got != "boom" {
		t.Fatalf("error message: got %q, want %q", got, "boom")
	}
}

func TestExitErrorf(t *testing.T) {
	err := exitErrorf(7, "bad %s", "wolf")
	ec, ok := err.(exitCoder)
	if !ok {
		t.Fatalf("error does not implement exitCoder: %T", err)
	}
	if got := ec.ExitCode(); got != 7 {
		t.Fatalf("exit code: got %d, want 7", got)
	}
	if got := err.Error(); got != "bad wolf" {
		t.Fatalf("error message: got %q, want %q", got, "bad wolf")
	}
}

func TestNewMapCmdStructure(t *testing.T) {
	cmd := newMapCmd()
	if cmd.Use != "map" {
		t.Fatalf("use: got %q, want 'map'", cmd.Use)
	}

	for _, flag := range []string{"region-dir", "region-file", "x", "y", "z"} {
		if f := cmd.PersistentFlags().Lookup(flag); f == nil {
			t.Fatalf("persistent flag %q not registered", flag)
		}
	}

	wantSubs := map[string]bool{"get": true, "create": true, "delete": true}
	for _, sub := range cmd.Commands() {
		delete(wantSubs, sub.Name())
	}
	if len(wantSubs) != 0 {
		t.Fatalf("missing subcommands: %v", wantSubs)
	}
}

