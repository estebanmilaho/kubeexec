package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func CurrentContext() (string, error) {
	cmd := exec.Command("kubectl", "config", "current-context")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("kubectl config current-context failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func CurrentNamespace() (string, error) {
	cmd := exec.Command("kubectl", "config", "view", "--minify", "--output", "jsonpath={..namespace}")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("kubectl config view failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func GetPods(namespace, selector string) ([]string, error) {
	args := []string{"get", "pods", "-o", "name"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if selector != "" {
		args = append(args, "-l", selector)
	}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var pods []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		pods = append(pods, strings.TrimPrefix(l, "pod/"))
	}
	return pods, nil
}

func GetPodContainers(namespace, pod string) ([]string, string, error) {
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
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.Output()
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

func ExecPod(namespace, pod, container string) error {
	args := []string{"exec", "-it"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, pod)
	if container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, "--", "sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh")
	cmd := exec.Command("kubectl", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
