package log_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gearlog "gearup/internal/log"
)

func TestCreate_OpensFileWritesHeader(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	lf, err := gearlog.Create("backend")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer lf.Close()

	if lf.Path() == "" {
		t.Fatal("Path() is empty")
	}
	if !strings.Contains(lf.Path(), filepath.Join("gearup", "logs")) {
		t.Errorf("Path() = %q, want to contain gearup/logs", lf.Path())
	}
	if !strings.Contains(lf.Path(), "backend") {
		t.Errorf("Path() = %q, want to contain recipe name", lf.Path())
	}

	if _, err := lf.Writer().Write([]byte("hello log\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := lf.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	data, err := os.ReadFile(lf.Path())
	if err != nil {
		t.Fatalf("readfile: %v", err)
	}
	if !strings.Contains(string(data), "recipe: backend") {
		t.Errorf("log missing header, got: %q", string(data))
	}
	if !strings.Contains(string(data), "hello log") {
		t.Errorf("log missing written content, got: %q", string(data))
	}
}

func TestCreate_FallsBackToHomeWhenXDGUnset(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", dir)

	lf, err := gearlog.Create("frontend")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer lf.Close()

	want := filepath.Join(dir, ".local", "state", "gearup", "logs")
	if !strings.HasPrefix(lf.Path(), want) {
		t.Errorf("Path() = %q, want prefix %q", lf.Path(), want)
	}
}
