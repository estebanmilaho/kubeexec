package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Run(contextArg, namespace, container, selector, podArg string) error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found")
	}

	context := contextArg
	if contextArg != "" {
		resolved, err := resolveContext(contextArg)
		if err != nil {
			return err
		}
		context = resolved
	}
	if context == "" {
		if namespace == "" {
			var err error
			context, err = CurrentContext()
			if err != nil {
				return err
			}
			if context == "" {
				return fmt.Errorf("no kubernetes context is set")
			}
		} else {
			if ctx, err := CurrentContext(); err == nil {
				context = ctx
			}
		}
	}

	if namespace == "" {
		var err error
		namespace, err = CurrentNamespace(context)
		if err != nil {
			return err
		}
		if namespace == "" {
			namespace = "default"
		}
	}

	pods, err := GetPods(context, namespace, selector)
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("no pods found")
	}

	pod := ""
	if podArg == "" {
		header := buildPodHeader(context, namespace, selector, "")
		if err := ensureFzf(); err != nil {
			return err
		}
		choice, err := ChooseWithFzf(pods, header)
		if err != nil {
			return err
		}
		if choice == "" {
			return fmt.Errorf("no pod selected")
		}
		pod = choice
	} else if contains(pods, podArg) {
		pod = podArg
	} else {
		matches := filterByQuery(pods, podArg)
		if len(matches) == 0 {
			return fmt.Errorf("no pods match %q", podArg)
		}
		if len(matches) == 1 {
			pod = matches[0]
		} else {
			header := buildPodHeader(context, namespace, selector, "pod: "+podArg)
			if err := ensureFzf(); err != nil {
				return err
			}
			choice, err := ChooseWithFzf(matches, header)
			if err != nil {
				return err
			}
			if choice == "" {
				return fmt.Errorf("no pod selected")
			}
			pod = choice
		}
	}

	containers, defaultContainer, err := GetPodContainers(context, namespace, pod)
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		return fmt.Errorf("no containers found in pod %q", pod)
	}

	if container != "" {
		if !contains(containers, container) {
			return fmt.Errorf("container %q not found in pod %q (available: %s)", container, pod, strings.Join(containers, ", "))
		}
		return ExecPod(context, namespace, pod, container)
	}

	if len(containers) == 1 {
		return ExecPod(context, namespace, pod, containers[0])
	}
	if defaultContainer != "" {
		fmt.Fprintf(os.Stderr, "note: pod has multiple containers (%s); using default %q. Use -c to select another.\n", strings.Join(containers, ", "), defaultContainer)
		return ExecPod(context, namespace, pod, defaultContainer)
	}

	if err := ensureFzf(); err != nil {
		return err
	}
	containerChoice, err := ChooseWithFzf(containers, fmt.Sprintf("pod: %s", pod))
	if err != nil {
		return err
	}
	if containerChoice == "" {
		return fmt.Errorf("no container selected")
	}

	return ExecPod(context, namespace, pod, containerChoice)
}

func contains(items []string, item string) bool {
	for _, it := range items {
		if it == item {
			return true
		}
	}
	return false
}

func filterByQuery(items []string, query string) []string {
	var matches []string
	for _, item := range items {
		if strings.Contains(item, query) {
			matches = append(matches, item)
		}
	}
	return matches
}

func buildPodHeader(context, namespace, selector, podQuery string) string {
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
	if podQuery != "" {
		parts = append(parts, podQuery)
	}
	return strings.Join(parts, "  ")
}

func resolveContext(query string) (string, error) {
	contexts, err := GetContexts()
	if err != nil {
		return "", err
	}
	if len(contexts) == 0 {
		return "", fmt.Errorf("no kubernetes contexts found")
	}
	if contains(contexts, query) {
		return query, nil
	}
	matches := filterByQuery(contexts, query)
	if len(matches) == 0 {
		return "", fmt.Errorf("no contexts match %q", query)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if err := ensureFzf(); err != nil {
		return "", err
	}
	choice, err := ChooseWithFzf(matches, "context query: "+query)
	if err != nil {
		return "", err
	}
	if choice == "" {
		return "", fmt.Errorf("no context selected")
	}
	return choice, nil
}

func ensureFzf() error {
	if _, err := exec.LookPath("fzf"); err != nil {
		return fmt.Errorf("fzf not found")
	}
	return nil
}
