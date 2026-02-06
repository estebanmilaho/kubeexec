package cmdutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	kubectlTimeoutDefault = 5 * time.Second
	kubectlTimeoutPods    = 15 * time.Second
)

func CurrentContext() (string, error) {
	out, err := runKubectl(kubectlTimeoutDefault, "config", "current-context")
	if err != nil {
		return "", fmt.Errorf("kubectl config current-context failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func GetContexts() ([]string, error) {
	out, err := runKubectl(kubectlTimeoutDefault, "config", "get-contexts", "-o", "name")
	if err != nil {
		return nil, fmt.Errorf("kubectl config get-contexts failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var contexts []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		contexts = append(contexts, l)
	}
	return contexts, nil
}

type PodItem struct {
	Name    string
	Ready   string
	Status  string
	Display string
}

func CurrentNamespace(context string) (string, error) {
	args := []string{"config", "view", "--minify", "--output", "jsonpath={..namespace}"}
	args = kubectlArgs(context, args...)
	out, err := runKubectl(kubectlTimeoutDefault, args...)
	if err != nil {
		return "", fmt.Errorf("kubectl config view failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func GetPods(context, namespace, selector string) ([]PodItem, error) {
	args := []string{"get", "pods", "-o", "custom-columns=NAME:.metadata.name,READY:.status.containerStatuses[*].ready,STATUS:.status.phase", "--no-headers"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if selector != "" {
		args = append(args, "-l", selector)
	}
	args = kubectlArgs(context, args...)
	out, err := runKubectl(kubectlTimeoutPods, args...)
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var pods []PodItem
	maxName := 0
	maxReady := 0
	maxStatus := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		readyRaw := ""
		status := ""
		if len(fields) > 1 {
			readyRaw = fields[1]
		}
		if len(fields) > 2 {
			status = fields[2]
		}
		ready := formatReady(readyRaw)
		pods = append(pods, PodItem{
			Name:   name,
			Ready:  ready,
			Status: status,
		})
		if len(name) > maxName {
			maxName = len(name)
		}
		if len(ready) > maxReady {
			maxReady = len(ready)
		}
		if len(status) > maxStatus {
			maxStatus = len(status)
		}
	}
	for i := range pods {
		pods[i].Display = fmt.Sprintf("%-*s  %-*s  %-*s", maxName, pods[i].Name, maxReady, pods[i].Ready, maxStatus, pods[i].Status)
	}
	return pods, nil
}

func GetPodContainers(context, namespace, pod string) ([]string, string, error) {
	args := []string{
		"get",
		"pod",
		pod,
		"-o",
		"jsonpath={.metadata.annotations.kubectl\\.kubernetes\\.io/default-container}{\"\\n\"}{range .spec.containers[*]}{.name}{\"\\n\"}{end}",
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = kubectlArgs(context, args...)
	out, err := runKubectl(kubectlTimeoutDefault, args...)
	if err != nil {
		return nil, "", fmt.Errorf("kubectl get pod failed: %w", err)
	}

	raw := strings.TrimRight(string(out), "\n")
	lines := strings.Split(raw, "\n")
	defaultContainer := ""
	if len(lines) > 0 {
		defaultContainer = strings.TrimSpace(lines[0])
	}

	var containers []string
	if len(lines) > 1 {
		for _, l := range lines[1:] {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			containers = append(containers, l)
		}
	}

	if defaultContainer != "" && !contains(containers, defaultContainer) {
		defaultContainer = ""
	}
	if defaultContainer == "" && len(containers) == 1 {
		defaultContainer = containers[0]
	}

	return containers, defaultContainer, nil
}

func ExecPod(context, namespace, pod, container string, command []string, nonInteractive bool) error {
	args := ExecArgs(context, namespace, pod, container, command, nonInteractive)
	cmd := exec.Command("kubectl", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func kubectlArgs(context string, args ...string) []string {
	if context == "" {
		return args
	}
	return append([]string{"--context", context}, args...)
}

func runKubectl(timeout time.Duration, args ...string) ([]byte, error) {
	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			msg := strings.TrimSpace(stderr.String())
			if msg != "" {
				return nil, fmt.Errorf("kubectl timed out after %s: %s", timeout, msg)
			}
			return nil, fmt.Errorf("kubectl timed out after %s", timeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func ExecArgs(context, namespace, pod, container string, command []string, nonInteractive bool) []string {
	args := []string{"exec"}
	if !nonInteractive {
		args = append(args, "-i")
		if isTerminal(os.Stdin) && isTerminal(os.Stdout) {
			args = append(args, "-t")
		}
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, pod)
	if container != "" {
		args = append(args, "-c", container)
	}
	if len(command) > 0 {
		args = append(args, "--")
		args = append(args, command...)
	} else {
		args = append(args, "--", "sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh")
	}
	return kubectlArgs(context, args...)
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func formatReady(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "<none>" || raw == "-" {
		return "-"
	}
	if strings.Contains(raw, "/") {
		return raw
	}
	parts := strings.Split(raw, ",")
	total := 0
	ready := 0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		total++
		if strings.EqualFold(p, "true") {
			ready++
		} else if strings.EqualFold(p, "false") {
			continue
		} else {
			if _, err := strconv.Atoi(p); err == nil {
				// If it's already numeric, just return the raw value.
				return raw
			}
		}
	}
	if total == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d", ready, total)
}
