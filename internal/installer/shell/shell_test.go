package shell_test

import (
	"context"
	"testing"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer/shell"
)

func TestShell_CheckInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("test -f /opt/thing").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing", Install: "touch /opt/thing"}
	ok, err := inst.Check(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true")
	}
}

func TestShell_CheckNotInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("test -f /opt/thing").Return(gearexec.Result{ExitCode: 1}, nil)

	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing", Install: "touch /opt/thing"}
	ok, err := inst.Check(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("installed = true, want false")
	}
}

func TestShell_MissingCheck(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Install: "touch /opt/thing"}
	_, err := inst.Check(context.Background(), step)
	if err == nil {
		t.Error("want error for missing check, got nil")
	}
}

func TestShell_MissingInstall(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing"}
	err := inst.Install(context.Background(), step)
	if err == nil {
		t.Error("want error for missing install, got nil")
	}
}

func TestShell_InstallSuccess(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("touch /opt/thing").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing", Install: "touch /opt/thing"}
	if err := inst.Install(context.Background(), step); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestShell_InstallFailure(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("touch /opt/thing").Return(gearexec.Result{ExitCode: 1, Stderr: "permission denied"}, nil)

	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing", Install: "touch /opt/thing"}
	if err := inst.Install(context.Background(), step); err == nil {
		t.Error("want error for failed install, got nil")
	}
}
