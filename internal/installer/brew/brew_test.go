package brew_test

import (
	"context"
	"testing"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer/brew"
)

func TestBrew_CheckInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 0, Stdout: "/opt/homebrew/Cellar/jq/1.7"}, nil)

	inst := &brew.Installer{Runner: f}
	ok, err := inst.Check(context.Background(), config.Step{Name: "jq", Type: "brew", Formula: "jq"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true")
	}
}

func TestBrew_CheckNotInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 1, Stderr: "Error: No such keg"}, nil)

	inst := &brew.Installer{Runner: f}
	ok, err := inst.Check(context.Background(), config.Step{Name: "jq", Type: "brew", Formula: "jq"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("installed = true, want false")
	}
}

func TestBrew_CheckMissingFormula(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &brew.Installer{Runner: f}
	_, err := inst.Check(context.Background(), config.Step{Name: "bad", Type: "brew"})
	if err == nil {
		t.Error("want error for missing formula, got nil")
	}
}

func TestBrew_InstallSuccess(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew install jq").Return(gearexec.Result{ExitCode: 0, Stdout: "==> Pouring jq"}, nil)

	inst := &brew.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{Name: "jq", Type: "brew", Formula: "jq"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBrew_InstallFailure(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew install missing-pkg").Return(gearexec.Result{ExitCode: 1, Stderr: "Error: No available formula"}, nil)

	inst := &brew.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{Name: "missing", Type: "brew", Formula: "missing-pkg"})
	if err == nil {
		t.Error("want error for failed install, got nil")
	}
}
