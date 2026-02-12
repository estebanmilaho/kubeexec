package cmdutil

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	confirmContextEnvVar   = "KUBEEXEC_CONFIRM_CONTEXT"
	nonInteractiveEnvVar   = "KUBEEXEC_NON_INTERACTIVE"
	ignoreFzfEnvVar        = "KUBEEXEC_IGNORE_FZF"
	confirmBoolValueHint   = "true/True/1/on/ON/false/False/0/off/OFF"
	confirmConfigValueHint = "true/false (TOML boolean)"
	confirmConfigFilename  = ".config/kubeexec/kubeexec.toml"
)

var defaultConfirmContextKeywords = []string{"prod", "production", "live"}

func ResolveConfirmContext(flagSet bool, flagValue bool) (bool, error) {
	return resolveBoolSetting(flagSet, flagValue, confirmContextEnvVar, "confirm-context")
}

func ResolveNonInteractive(flagSet bool, flagValue bool) (bool, error) {
	return resolveBoolSetting(flagSet, flagValue, nonInteractiveEnvVar, "non-interactive")
}

func ResolveIgnoreFzf(flagSet bool, flagValue bool) (bool, error) {
	return resolveBoolSetting(flagSet, flagValue, ignoreFzfEnvVar, "ignore-fzf")
}

func resolveBoolSetting(flagSet bool, flagValue bool, envVar string, configKey string) (bool, error) {
	if flagSet {
		return flagValue, nil
	}
	if val, ok := os.LookupEnv(envVar); ok {
		parsed, ok := ParseConfirmBool(val)
		if !ok {
			return false, fmt.Errorf("invalid %s value %q (use %s)", envVar, val, confirmBoolValueHint)
		}
		return parsed, nil
	}
	settings, err := loadConfigSettings()
	if err != nil {
		return false, err
	}
	if configKey == "confirm-context" && settings.ConfirmContext != nil {
		return *settings.ConfirmContext, nil
	}
	if configKey == "non-interactive" && settings.NonInteractive != nil {
		return *settings.NonInteractive, nil
	}
	if configKey == "ignore-fzf" && settings.IgnoreFzf != nil {
		return *settings.IgnoreFzf, nil
	}
	return false, nil
}

type configSettings struct {
	ConfirmContext         *bool    `toml:"confirm-context"`
	NonInteractive         *bool    `toml:"non-interactive"`
	IgnoreFzf              *bool    `toml:"ignore-fzf"`
	ConfirmContextKeywords []string `toml:"confirm-context-keywords"`
}

func loadConfigSettings() (configSettings, error) {
	var settings configSettings
	path, err := kubeexecConfigPath()
	if err != nil {
		return settings, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil
		}
		return settings, fmt.Errorf("read config %s: %w", path, err)
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return settings, fmt.Errorf("config %s is empty (expected TOML booleans: %s)", path, confirmConfigValueHint)
	}
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&settings); err != nil {
		return settings, fmt.Errorf("parse config %s: %w", path, err)
	}
	return settings, nil
}

func kubeexecConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, confirmConfigFilename), nil
}

func ParseConfirmBool(value string) (bool, bool) {
	switch strings.TrimSpace(value) {
	case "true", "True", "1", "on", "ON":
		return true, true
	case "false", "False", "0", "off", "OFF":
		return false, true
	default:
		return false, false
	}
}

func resolveConfirmContextKeywords() []string {
	settings, err := loadConfigSettings()
	if err == nil && len(settings.ConfirmContextKeywords) > 0 {
		if normalized := normalizeKeywords(settings.ConfirmContextKeywords); len(normalized) > 0 {
			return normalized
		}
	}
	return normalizeKeywords(defaultConfirmContextKeywords)
}

func confirmContextMatch(context, namespace string) bool {
	keywords := resolveConfirmContextKeywords()
	return containsKeyword(context, keywords) || containsKeyword(namespace, keywords)
}

func containsKeyword(value string, keywords []string) bool {
	lower := strings.ToLower(value)
	segments := segmentName(lower)
	for _, keyword := range keywords {
		for _, seg := range segments {
			if seg == keyword {
				return true
			}
		}
	}
	return false
}

func normalizeKeywords(keywords []string) []string {
	normalized := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(strings.ToLower(keyword))
		if keyword == "" {
			continue
		}
		normalized = append(normalized, keyword)
	}
	return normalized
}

// segmentName splits a context/namespace name into segments by common delimiters.
func segmentName(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == '/'
	})
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
