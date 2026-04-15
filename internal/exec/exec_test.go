package exec_test

import (
	"context"
	"strings"
	"testing"

	gearexec "gearup/internal/exec"
)

func TestShellRunner_TrueExits0(t *testing.T) {
	r := &gearexec.ShellRunner{}
	res, err := r.Run(context.Background(), "true")
	if err != nil {
		t.Fatalf("unexpected spawn error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
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

func TestShellRunner_CapturesStdout(t *testing.T) {
	r := &gearexec.ShellRunner{}
	res, err := r.Run(context.Background(), "echo hello-gearup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello-gearup") {
		t.Errorf("Stdout = %q, want contains hello-gearup", res.Stdout)
	}
}

func TestFakeRunner_ReturnsProgrammedResult(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 0, Stdout: "/opt/homebrew/Cellar/jq"}, nil)
	res, err := f.Run(context.Background(), "brew list --formula jq")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "Cellar/jq") {
		t.Errorf("Stdout = %q", res.Stdout)
	}
	if got := f.Calls(); len(got) != 1 || got[0] != "brew list --formula jq" {
		t.Errorf("Calls = %v", got)
	}
}

func TestFakeRunner_UnstubbedCommandFails(t *testing.T) {
	f := gearexec.NewFakeRunner()
	_, err := f.Run(context.Background(), "some unstubbed command")
	if err == nil {
		t.Error("want error for unstubbed command, got nil")
	}
}
