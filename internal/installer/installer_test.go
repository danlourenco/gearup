package installer_test

import (
	"context"
	"testing"

	"gearup/internal/config"
	"gearup/internal/installer"
)

type stubInstaller struct {
	name string
}

func (s *stubInstaller) Check(_ context.Context, _ config.Step) (bool, error) { return true, nil }
func (s *stubInstaller) Install(_ context.Context, _ config.Step) error       { return nil }

func TestRegistry_GetKnownType(t *testing.T) {
	reg := installer.Registry{"brew": &stubInstaller{name: "brew"}}
	got, err := reg.Get("brew")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("want non-nil Installer")
	}
}

func TestRegistry_GetUnknownType(t *testing.T) {
	reg := installer.Registry{}
	_, err := reg.Get("not-a-type")
	if err == nil {
		t.Error("want error for unknown step type, got nil")
	}
}
