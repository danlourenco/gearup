package exec_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	gearexec "gearup/internal/exec"
)

func TestShellRunner_Run_StreamsAndLogs(t *testing.T) {
	var stream, log bytes.Buffer
	r := &gearexec.ShellRunner{StreamOut: &stream, LogOut: &log}
	res, err := r.Run(context.Background(), "echo hello-gearup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if !strings.Contains(stream.String(), "hello-gearup") {
		t.Errorf("stream missing output: %q", stream.String())
	}
	if !strings.Contains(log.String(), "hello-gearup") {
		t.Errorf("log missing output: %q", log.String())
	}
}

func TestShellRunner_RunQuiet_LogsButDoesNotStream(t *testing.T) {
	var stream, log bytes.Buffer
	r := &gearexec.ShellRunner{StreamOut: &stream, LogOut: &log}
	res, err := r.RunQuiet(context.Background(), "echo should-be-quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if stream.Len() != 0 {
		t.Errorf("stream should be empty, got: %q", stream.String())
	}
	if !strings.Contains(log.String(), "should-be-quiet") {
		t.Errorf("log missing output: %q", log.String())
	}
}

func TestShellRunner_Run_NilSinksAreSafe(t *testing.T) {
	r := &gearexec.ShellRunner{}
	res, err := r.Run(context.Background(), "echo ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Stdout, "ok") {
		t.Errorf("captured stdout should still contain echo output: %q", res.Stdout)
	}
}

func TestShellRunner_FalseExitsNonzero(t *testing.T) {
	r := &gearexec.ShellRunner{}
	res, err := r.Run(context.Background(), "false")
	if err != nil {
		t.Fatalf("unexpected spawn error: %v", err)
	}
	if res.ExitCode == 0 {
		t.Error("ExitCode = 0, want non-zero")
	}
}

func TestFakeRunner_Run_ReturnsProgrammedResult(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 0, Stdout: "/opt/homebrew/Cellar/jq"}, nil)
	res, err := f.Run(context.Background(), "brew list --formula jq")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if got := f.Calls(); len(got) != 1 || got[0].Cmd != "brew list --formula jq" || got[0].Quiet {
		t.Errorf("Calls = %+v, want one non-quiet call to brew list", got)
	}
}

func TestFakeRunner_RunQuiet_MarksCallAsQuiet(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 0}, nil)
	_, err := f.RunQuiet(context.Background(), "brew list --formula jq")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := f.Calls()
	if len(got) != 1 {
		t.Fatalf("Calls len = %d, want 1", len(got))
	}
	if !got[0].Quiet {
		t.Errorf("Calls[0].Quiet = false, want true")
	}
}

func TestFakeRunner_UnstubbedCommandFails(t *testing.T) {
	f := gearexec.NewFakeRunner()
	_, err := f.Run(context.Background(), "some unstubbed command")
	if err == nil {
		t.Error("want error for unstubbed command, got nil")
	}
}
