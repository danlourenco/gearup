package gitclone_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer/gitclone"
)

func TestGitClone_CheckDestExists(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "repo")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	inst := &gitclone.Installer{}
	ok, err := inst.Check(context.Background(), config.Step{
		Name: "dotfiles", Type: "git-clone",
		Repo: "https://github.com/example/dotfiles.git", Dest: dest,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true (dir exists)")
	}
}

func TestGitClone_CheckDestMissing(t *testing.T) {
	inst := &gitclone.Installer{}
	ok, err := inst.Check(context.Background(), config.Step{
		Name: "dotfiles", Type: "git-clone",
		Repo: "https://github.com/example/dotfiles.git",
		Dest: "/nonexistent/path/dotfiles",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("installed = true, want false (dir missing)")
	}
}

func TestGitClone_CheckMissingDest(t *testing.T) {
	inst := &gitclone.Installer{}
	_, err := inst.Check(context.Background(), config.Step{
		Name: "bad", Type: "git-clone", Repo: "https://example.com/repo.git",
	})
	if err == nil {
		t.Error("want error for missing dest, got nil")
	}
}

func TestGitClone_InstallSuccess(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "sub", "repo")

	f := gearexec.NewFakeRunner()
	f.On("git clone https://example.com/repo.git "+dest).Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &gitclone.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{
		Name: "test", Type: "git-clone",
		Repo: "https://example.com/repo.git", Dest: dest,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitClone_InstallWithRef(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "repo")

	f := gearexec.NewFakeRunner()
	f.On("git clone --branch main https://example.com/repo.git "+dest).Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &gitclone.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{
		Name: "test", Type: "git-clone",
		Repo: "https://example.com/repo.git", Dest: dest, Ref: "main",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	got := f.Calls()
	if len(got) != 1 || got[0].Cmd != "git clone --branch main https://example.com/repo.git "+dest {
		t.Errorf("Calls = %+v", got)
	}
}

func TestGitClone_InstallFailure(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("git clone https://example.com/repo.git /tmp/fail").Return(gearexec.Result{ExitCode: 128, Stderr: "fatal: repo not found"}, nil)

	inst := &gitclone.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{
		Name: "test", Type: "git-clone",
		Repo: "https://example.com/repo.git", Dest: "/tmp/fail",
	})
	if err == nil {
		t.Error("want error for failed clone, got nil")
	}
}

func TestGitClone_InstallMissingRepo(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &gitclone.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{
		Name: "bad", Type: "git-clone", Dest: "/tmp/x",
	})
	if err == nil {
		t.Error("want error for missing repo, got nil")
	}
}
