package cmdutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfirmContextFromConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := filepath.Join(dir, ".config", "kubeexec", "kubeexec.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("confirm-context = true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	val, err := ResolveConfirmContext(false, false)
	if err != nil {
		t.Fatalf("ResolveConfirmContext error: %v", err)
	}
	if !val {
		t.Fatalf("expected confirm-context true from config")
	}
}

func TestResolveNonInteractiveFromConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := filepath.Join(dir, ".config", "kubeexec", "kubeexec.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("non-interactive = true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	val, err := ResolveNonInteractive(false, false)
	if err != nil {
		t.Fatalf("ResolveNonInteractive error: %v", err)
	}
	if !val {
		t.Fatalf("expected non-interactive true from config")
	}
}

func TestEnvOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := filepath.Join(dir, ".config", "kubeexec", "kubeexec.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("confirm-context = true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(confirmContextEnvVar, "false")
	val, err := ResolveConfirmContext(false, false)
	if err != nil {
		t.Fatalf("ResolveConfirmContext error: %v", err)
	}
	if val {
		t.Fatalf("expected env to override config")
	}
}

func TestInvalidTomlValue(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := filepath.Join(dir, ".config", "kubeexec", "kubeexec.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("confirm-context = \"true\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	_, err := ResolveConfirmContext(false, false)
	if err == nil {
		t.Fatalf("expected error for non-boolean TOML value")
	}
}

func TestUnknownTomlKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := filepath.Join(dir, ".config", "kubeexec", "kubeexec.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("unknown = true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	_, err := ResolveConfirmContext(false, false)
	if err == nil {
		t.Fatalf("expected error for unknown TOML key")
	}
}
