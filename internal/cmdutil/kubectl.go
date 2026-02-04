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

func GetPods() ([]string, error) {
    cmd := exec.Command("kubectl", "get", "pods", "-o", "name")
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

func ExecPod(pod string) error {
    shell := "bash"
    check := exec.Command("kubectl", "exec", "-it", pod, "--", "sh", "-c", "command -v bash >/dev/null 2>&1")
    check.Stdin, check.Stdout, check.Stderr = os.Stdin, os.Stdout, os.Stderr
    if err := check.Run(); err != nil {
        shell = "sh"
    }

    cmd := exec.Command("kubectl", "exec", "-it", pod, "--", shell)
    cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
    return cmd.Run()
}
