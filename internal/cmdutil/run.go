package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Run(contextArg, namespace, container, selector, podArg string, dryRun bool, contextRequested bool, confirmContext bool) error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found")
	}

	context := contextArg
	if contextRequested {
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
			ctx, err := CurrentContext()
			if err != nil {
				return err
			}
			context = ctx
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
		choice, err := ChooseWithFzf(podDisplays(pods), header)
		if err != nil {
			return err
		}
		if choice == "" {
			return fmt.Errorf("no pod selected")
		}
		pod = podNameFromChoice(choice)
		if pod == "" {
			return fmt.Errorf("no pod selected")
		}
	} else if podExists(pods, podArg) {
		pod = podArg
	} else {
		matches := filterPodsByQuery(pods, podArg)
		if len(matches) == 0 {
			return fmt.Errorf("no pods match %q", podArg)
		}
		if len(matches) == 1 {
			pod = matches[0].Name
		} else {
			header := buildPodHeader(context, namespace, selector, "pod: "+podArg)
			if err := ensureFzf(); err != nil {
				return err
			}
			choice, err := ChooseWithFzf(podDisplays(matches), header)
			if err != nil {
				return err
			}
			if choice == "" {
				return fmt.Errorf("no pod selected")
			}
			pod = podNameFromChoice(choice)
			if pod == "" {
				return fmt.Errorf("no pod selected")
			}
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
		return execOrPrint(context, namespace, pod, container, dryRun, confirmContext)
	}

	if len(containers) == 1 {
		return execOrPrint(context, namespace, pod, containers[0], dryRun, confirmContext)
	}
	if defaultContainer != "" {
		fmt.Fprintf(os.Stderr, "note: pod has multiple containers (%s); using default %q. Use -c to select another.\n", strings.Join(containers, ", "), defaultContainer)
		return execOrPrint(context, namespace, pod, defaultContainer, dryRun, confirmContext)
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

	return execOrPrint(context, namespace, pod, containerChoice, dryRun, confirmContext)
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

func filterPodsByQuery(pods []PodItem, query string) []PodItem {
	var matches []PodItem
	for _, pod := range pods {
		if strings.Contains(pod.Name, query) {
			matches = append(matches, pod)
		}
	}
	return matches
}

func podExists(pods []PodItem, name string) bool {
	for _, pod := range pods {
		if pod.Name == name {
			return true
		}
	}
	return false
}

func podDisplays(pods []PodItem) []string {
	displays := make([]string, 0, len(pods))
	for _, pod := range pods {
		if pod.Display != "" {
			displays = append(displays, pod.Display)
			continue
		}
		displays = append(displays, pod.Name)
	}
	return displays
}

func podNameFromChoice(choice string) string {
	fields := strings.Fields(choice)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
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
	if query == "" {
		if err := ensureFzf(); err != nil {
			return "", err
		}
		choice, err := ChooseWithFzf(contexts, "select context")
		if err != nil {
			return "", err
		}
		if choice == "" {
			return "", fmt.Errorf("no context selected")
		}
		return choice, nil
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

func execOrPrint(context, namespace, pod, container string, dryRun bool, confirmContext bool) error {
	if dryRun {
		args := ExecArgs(context, namespace, pod, container)
		fmt.Fprintln(os.Stdout, "kubectl "+strings.Join(args, " "))
		return nil
	}
	if confirmContext && confirmContextMatch(context, namespace) {
		if err := confirmContextPrompt(context, namespace); err != nil {
			return err
		}
	}
	return ExecPod(context, namespace, pod, container)
}
