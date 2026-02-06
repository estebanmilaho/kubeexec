package cmdutil

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

func ChooseWithFzf(items []string, header string) (string, error) {
	args := []string{"--ansi", "--no-preview"}
	if header != "" {
		args = append(args, "--header", header)
	}
	cmd := exec.Command("fzf", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n") + "\n")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		// fzf returns non-zero on cancel or no matches; surface real errors.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			switch exitErr.ExitCode() {
			case 1, 130:
				return "", nil
			}
		}
		return "", err
	}

	choiceBytes, err := io.ReadAll(&stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(choiceBytes)), nil
}
