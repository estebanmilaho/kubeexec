package cmdutil

import (
	"strings"
	"testing"
)

func TestExecArgsDefaultShell(t *testing.T) {
	args := ExecArgs("ctx", "ns", "pod", "cont", nil, false)
	args = removeArg(args, "-t")

	expectedSuffix := []string{"--", "sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"}
	if !hasSubsequence(args, expectedSuffix) {
		t.Fatalf("expected default shell command, got %v", args)
	}
	if !containsArg(args, "-i") {
		t.Fatalf("expected -i when nonInteractive=false, got %v", args)
	}
}

func TestExecArgsCommandOverride(t *testing.T) {
	command := []string{"ls", "-la", "/"}
	args := ExecArgs("ctx", "ns", "pod", "cont", command, true)
	args = removeArg(args, "-t")

	if containsArg(args, "-i") {
		t.Fatalf("did not expect -i when nonInteractive=true, got %v", args)
	}
	expected := []string{"--context", "ctx", "exec", "-n", "ns", "pod", "-c", "cont", "--", "ls", "-la", "/"}
	if !hasSubsequence(args, expected) {
		t.Fatalf("expected args to include %v, got %v", expected, args)
	}
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func removeArg(args []string, target string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == target {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func hasSubsequence(args []string, expected []string) bool {
	if len(expected) == 0 {
		return true
	}
	joined := strings.Join(args, "\x00")
	needle := strings.Join(expected, "\x00")
	return strings.Contains(joined, needle)
}

func TestFormatReady(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"empty string", "", "-"},
		{"none", "<none>", "-"},
		{"dash", "-", "-"},
		{"single true", "true", "1/1"},
		{"single false", "false", "0/1"},
		{"two true", "true,true", "2/2"},
		{"mixed booleans", "true,false", "1/2"},
		{"three mixed", "true,false,true", "2/3"},
		{"all false", "false,false", "0/2"},
		{"already fraction", "1/2", "1/2"},
		{"full fraction", "3/3", "3/3"},
		{"whitespace", "  true  ", "1/1"},
		{"comma with spaces", "true, false", "1/2"},
		{"numeric passthrough", "42", "42"},
		{"case insensitive true", "True", "1/1"},
		{"case insensitive false", "False", "0/1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatReady(tt.raw)
			if got != tt.want {
				t.Errorf("formatReady(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
