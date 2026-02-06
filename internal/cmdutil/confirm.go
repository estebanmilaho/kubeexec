package cmdutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const confirmContextEnvVar = "KUBEEXEC_CONFIRM_CONTEXT"

var confirmContextKeywords = []string{"prod", "production", "live"}

func ResolveConfirmContext(flagSet bool, flagValue bool) (bool, error) {
	if flagSet {
		return flagValue, nil
	}
	if val, ok := os.LookupEnv(confirmContextEnvVar); ok {
		parsed, ok := parseConfirmBool(val)
		if !ok {
			return false, fmt.Errorf("invalid %s value %q (use true/True/1/false/False/0)", confirmContextEnvVar, val)
		}
		return parsed, nil
	}
	path, err := confirmContextConfigPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read config %s: %w", path, err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return false, fmt.Errorf("config %s is empty (use true/True/1/false/False/0)", path)
	}
	parsed, ok := parseConfirmBool(value)
	if !ok {
		return false, fmt.Errorf("invalid value in %s: %q (use true/True/1/false/False/0)", path, value)
	}
	return parsed, nil
}

func confirmContextConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".config", "kubeexec"), nil
}

func parseConfirmBool(value string) (bool, bool) {
	switch strings.TrimSpace(value) {
	case "true", "True", "1", "on", "ON":
		return true, true
	case "false", "False", "0", "off", "OFF":
		return false, true
	default:
		return false, false
	}
}

func confirmContextMatch(context, namespace string) bool {
	return containsKeyword(context) || containsKeyword(namespace)
}

func containsKeyword(value string) bool {
	lower := strings.ToLower(value)
	for _, keyword := range confirmContextKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func confirmContextPrompt(context, namespace string) error {
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return fmt.Errorf("confirmation required but no TTY available")
	}
	expected := context + "/" + namespace
	fmt.Fprintf(os.Stderr, "confirm context %q namespace %q: type %q to continue: ", context, namespace, expected)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	if strings.TrimSpace(line) != expected {
		return fmt.Errorf("context confirmation failed")
	}
	return nil
}
