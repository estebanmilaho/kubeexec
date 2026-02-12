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

func TestSegmentName(t *testing.T) {
	tests := []struct {
		name string
		input string
		want []string
	}{
		{"dashes", "my-prod-cluster", []string{"my", "prod", "cluster"}},
		{"underscores", "my_prod_cluster", []string{"my", "prod", "cluster"}},
		{"dots", "my.prod.cluster", []string{"my", "prod", "cluster"}},
		{"slashes", "my/prod/cluster", []string{"my", "prod", "cluster"}},
		{"mixed", "my-prod_cluster.east/1", []string{"my", "prod", "cluster", "east", "1"}},
		{"no delimiters", "production", []string{"production"}},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentName(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("segmentName(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("segmentName(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConfirmContextMatch(t *testing.T) {
	// Use a temp HOME with no config so default keywords apply
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tests := []struct {
		name      string
		context   string
		namespace string
		want      bool
	}{
		{"prod in context", "my-prod-cluster", "default", true},
		{"production in context", "production-east", "default", true},
		{"live in namespace", "dev-cluster", "live", true},
		{"no match", "dev-cluster", "staging", false},
		{"reproduce does not match", "reproduce-bug", "test", false},
		{"olive does not match", "olive-garden", "default", false},
		{"prod in namespace", "dev", "qb-prod", true},
		{"both match", "qb-prod-6", "qb-prod", true},
		{"empty strings", "", "", false},
		{"production as namespace", "dev", "production", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := confirmContextMatch(tt.context, tt.namespace)
			if got != tt.want {
				t.Errorf("confirmContextMatch(%q, %q) = %v, want %v", tt.context, tt.namespace, got, tt.want)
			}
		})
	}
}

func TestConfirmContextKeywordsFromConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := filepath.Join(dir, ".config", "kubeexec", "kubeexec.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("confirm-context-keywords = [\"staging\", \"uat\"]\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	keywords := resolveConfirmContextKeywords()
	if len(keywords) != 2 {
		t.Fatalf("expected 2 keywords, got %d: %v", len(keywords), keywords)
	}
	if keywords[0] != "staging" || keywords[1] != "uat" {
		t.Errorf("expected [staging, uat], got %v", keywords)
	}
}

func TestConfirmContextKeywordsDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// No config file — should use defaults
	keywords := resolveConfirmContextKeywords()
	if len(keywords) != 3 {
		t.Fatalf("expected 3 default keywords, got %d: %v", len(keywords), keywords)
	}
	expected := []string{"prod", "production", "live"}
	for i, kw := range keywords {
		if kw != expected[i] {
			t.Errorf("default keyword[%d] = %q, want %q", i, kw, expected[i])
		}
	}
}
