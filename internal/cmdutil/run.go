package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Run(namespace, container, selector string) error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		return fmt.Errorf("fzf not found")
	}

	context := ""
	if namespace == "" {
		var err error
		context, err = CurrentContext()
		if err != nil {
			return err
		}
		if context == "" {
			return fmt.Errorf("no kubernetes context is set")
		}

		namespace, err = CurrentNamespace()
		if err != nil {
			return err
		}
		if namespace == "" {
			namespace = "default"
		}
	} else {
		if ctx, err := CurrentContext(); err == nil {
			context = ctx
		}
	}

	pods, err := GetPods(namespace, selector)
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("no pods found")
	}

	header := buildPodHeader(context, namespace, selector)
	choice, err := ChooseWithFzf(pods, header)
	if err != nil {
		return err
	}
	if choice == "" {
		return fmt.Errorf("no pod selected")
	}

	containers, defaultContainer, err := GetPodContainers(namespace, choice)
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		return fmt.Errorf("no containers found in pod %q", choice)
	}

	if container != "" {
		if !contains(containers, container) {
			return fmt.Errorf("container %q not found in pod %q (available: %s)", container, choice, strings.Join(containers, ", "))
		}
		return ExecPod(namespace, choice, container)
	}

	if len(containers) == 1 {
		return ExecPod(namespace, choice, containers[0])
	}
	if defaultContainer != "" {
		fmt.Fprintf(os.Stderr, "note: pod has multiple containers (%s); using default %q. Use -c to select another.\n", strings.Join(containers, ", "), defaultContainer)
		return ExecPod(namespace, choice, defaultContainer)
	}

	containerChoice, err := ChooseWithFzf(containers, fmt.Sprintf("pod: %s", choice))
	if err != nil {
		return err
	}
	if containerChoice == "" {
		return fmt.Errorf("no container selected")
	}

	return ExecPod(namespace, choice, containerChoice)
}

func contains(items []string, item string) bool {
	for _, it := range items {
		if it == item {
			return true
		}
	}
	return false
}

func buildPodHeader(context, namespace, selector string) string {
	var parts []string
	if context != "" {
		parts = append(parts, "context: "+context)
	}
	if namespace != "" {
		parts = append(parts, "namespace: "+namespace)
	}
	if selector != "" {
		parts = append(parts, "selector: "+selector)
	}
	return strings.Join(parts, "  ")
}
