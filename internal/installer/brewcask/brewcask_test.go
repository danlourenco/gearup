package brewcask_test

import (
	"context"
	"testing"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer/brewcask"
)

func TestBrewCask_CheckInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --cask iterm2").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &brewcask.Installer{Runner: f}
	ok, err := inst.Check(context.Background(), config.Step{Name: "iTerm2", Type: "brew-cask", Cask: "iterm2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true")
	}
}

func TestBrewCask_CheckNotInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --cask iterm2").Return(gearexec.Result{ExitCode: 1, Stderr: "Error: Cask iterm2 is not installed"}, nil)

	inst := &brewcask.Installer{Runner: f}
	ok, err := inst.Check(context.Background(), config.Step{Name: "iTerm2", Type: "brew-cask", Cask: "iterm2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("installed = true, want false")
	}
}

func TestBrewCask_CheckMissingCask(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &brewcask.Installer{Runner: f}
	_, err := inst.Check(context.Background(), config.Step{Name: "bad", Type: "brew-cask"})
	if err == nil {
		t.Error("want error for missing cask, got nil")
	}
}

func TestBrewCask_CheckUsesOverride(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("test -d /Applications/iTerm.app").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &brewcask.Installer{Runner: f}
	ok, err := inst.Check(context.Background(), config.Step{
		Name: "iTerm2", Type: "brew-cask", Cask: "iterm2",
		Check: "test -d /Applications/iTerm.app",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true")
	}
	if got := f.Calls(); len(got) != 1 || got[0].Cmd != "test -d /Applications/iTerm.app" {
		t.Errorf("Calls = %+v, want override command", got)
	}
}

func TestBrewCask_InstallSuccess(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew install --cask iterm2").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &brewcask.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{Name: "iTerm2", Type: "brew-cask", Cask: "iterm2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBrewCask_InstallFailure(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew install --cask badcask").Return(gearexec.Result{ExitCode: 1, Stderr: "Error: Cask badcask not found"}, nil)

	inst := &brewcask.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{Name: "bad", Type: "brew-cask", Cask: "badcask"})
	if err == nil {
		t.Error("want error for failed install, got nil")
	}
}
