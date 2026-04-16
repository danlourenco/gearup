package curlpipe_test

import (
	"context"
	"strings"
	"testing"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer/curlpipe"
)

func TestCurlPipe_CheckInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("command -v thing").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{
		Name:  "thing",
		Type:  "curl-pipe-sh",
		URL:   "https://example.com/install.sh",
		Check: "command -v thing",
	}
	ok, err := inst.Check(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true")
	}
}

func TestCurlPipe_CheckNotInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("command -v thing").Return(gearexec.Result{ExitCode: 1}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", URL: "https://example.com/install.sh", Check: "command -v thing"}
	ok, err := inst.Check(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("installed = true, want false")
	}
}

func TestCurlPipe_MissingCheck(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", URL: "https://example.com/install.sh"}
	_, err := inst.Check(context.Background(), step)
	if err == nil {
		t.Error("want error for missing check, got nil")
	}
}

func TestCurlPipe_MissingURL(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", Check: "command -v thing"}
	err := inst.Install(context.Background(), step)
	if err == nil {
		t.Error("want error for missing url, got nil")
	}
}

func TestCurlPipe_InstallDefaultsToBash(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("curl -fsSL https://example.com/install.sh | bash").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", URL: "https://example.com/install.sh", Check: "command -v thing"}
	if err := inst.Install(context.Background(), step); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCurlPipe_InstallCustomShellAndArgs(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("curl -fsSL https://example.com/install.sh | sh -s -- --quiet").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{
		Name:  "thing",
		Type:  "curl-pipe-sh",
		URL:   "https://example.com/install.sh",
		Shell: "sh",
		Args:  []string{"--quiet"},
		Check: "command -v thing",
	}
	if err := inst.Install(context.Background(), step); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	got := f.Calls()
	if len(got) != 1 || !strings.Contains(got[0].Cmd, "| sh -s -- --quiet") {
		t.Errorf("Calls = %+v", got)
	}
}

func TestCurlPipe_InstallFailure(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("curl -fsSL https://example.com/install.sh | bash").Return(gearexec.Result{ExitCode: 1, Stderr: "curl: connection refused"}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", URL: "https://example.com/install.sh", Check: "command -v thing"}
	if err := inst.Install(context.Background(), step); err == nil {
		t.Error("want error for failed install, got nil")
	}
}
