package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RunOptions struct {
	Context          string
	Namespace        string
	Container        string
	Selector         string
	Pod              string
	Command          []string
	DryRun           bool
	ContextRequested bool
	ConfirmContext   bool
	NonInteractive   bool
	IgnoreFzf        bool
	AllNamespaces    bool
}

func Run(opts RunOptions) error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found")
	}

	context := opts.Context
	namespace := opts.Namespace
	container := opts.Container
	selector := opts.Selector
	podArg := opts.Pod
	command := opts.Command
	dryRun := opts.DryRun
	confirmContext := opts.ConfirmContext
	nonInteractive := opts.NonInteractive
	ignoreFzf := opts.IgnoreFzf
	allNamespaces := opts.AllNamespaces

	if opts.ContextRequested {
		resolved, err := resolveContext(context, ignoreFzf)
		if err != nil {
			return err
		}
		context = resolved
	}
	if context == "" {
		var err error
		context, err = CurrentContext()
		if err != nil {
			return err
		}
		if context == "" && namespace == "" {
			return fmt.Errorf("no kubernetes context is set")
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

	if allNamespaces {
		namespace = ""
	}

	pods, err := GetPods(context, namespace, selector, allNamespaces)
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("no pods found")
	}

	pod := ""
	podNamespace := namespace
	if podArg == "" {
		if ignoreFzf {
			return fmt.Errorf("pod not specified and fzf is disabled; provide a pod name or enable fzf")
		}
		if err := checkFzf(); err != nil {
			return err
		}
		header := buildPodHeader(context, namespace, selector, "", allNamespaces)
		choice, err := ChooseWithFzf(podDisplays(pods), header)
		if err != nil {
			return err
		}
		if choice == "" {
			return fmt.Errorf("no pod selected")
		}
		selected, ok := podFromChoice(pods, choice)
		if !ok {
			return fmt.Errorf("no pod selected")
		}
		pod = selected.Name
		podNamespace = selected.Namespace
	} else if allNamespaces && strings.Contains(podArg, "/") {
		ns, name, ok := splitPodNamespaceArg(podArg)
		if !ok {
			return fmt.Errorf("invalid pod argument %q (expected namespace/pod)", podArg)
		}
		if !podExistsInNamespace(pods, ns, name) {
			return fmt.Errorf("pod %q not found in namespace %q", name, ns)
		}
		pod = name
		podNamespace = ns
	} else if podExists(pods, podArg) && !allNamespaces {
		pod = podArg
	} else {
		matches := filterPodsByQuery(pods, podArg)
		if len(matches) == 0 {
			return fmt.Errorf("no pods match %q", podArg)
		}
		if len(matches) == 1 {
			pod = matches[0].Name
			podNamespace = matches[0].Namespace
		} else {
			if ignoreFzf {
				if allNamespaces {
					return fmt.Errorf("pod query %q matches multiple entries and fzf is disabled; provide namespace/pod or enable fzf", podArg)
				}
				return fmt.Errorf("pod query %q matches multiple entries and fzf is disabled; provide a full pod name or enable fzf", podArg)
			}
			if err := checkFzf(); err != nil {
				return err
			}
			header := buildPodHeader(context, namespace, selector, "pod: "+podArg, allNamespaces)
			choice, err := ChooseWithFzf(podDisplays(matches), header)
			if err != nil {
				return err
			}
			if choice == "" {
				return fmt.Errorf("no pod selected")
			}
			selected, ok := podFromChoice(matches, choice)
			if !ok {
				return fmt.Errorf("no pod selected")
			}
			pod = selected.Name
			podNamespace = selected.Namespace
		}
	}

	if podNamespace == "" {
		podNamespace = namespace
	}
	containers, defaultContainer, err := GetPodContainers(context, podNamespace, pod)
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
		return execOrPrint(context, podNamespace, pod, container, command, dryRun, confirmContext, nonInteractive)
	}

	if len(containers) == 1 {
		return execOrPrint(context, podNamespace, pod, containers[0], command, dryRun, confirmContext, nonInteractive)
	}
	if defaultContainer != "" {
		fmt.Fprintf(os.Stderr, "note: pod has multiple containers (%s); using default %q. Use -c to select another.\n", strings.Join(containers, ", "), defaultContainer)
		return execOrPrint(context, podNamespace, pod, defaultContainer, command, dryRun, confirmContext, nonInteractive)
	}

	if ignoreFzf {
		return fmt.Errorf("pod %q has multiple containers and fzf is disabled; use -c to select a container or enable fzf", pod)
	}
	if err := checkFzf(); err != nil {
		return err
	}
	containerChoice, err := ChooseWithFzf(containers, fmt.Sprintf("pod: %s", pod))
	if err != nil {
		return err
	}
	if containerChoice == "" {
		return fmt.Errorf("no container selected")
	}

	return execOrPrint(context, podNamespace, pod, containerChoice, command, dryRun, confirmContext, nonInteractive)
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

func podFromChoice(pods []PodItem, choice string) (PodItem, bool) {
	trimmed := strings.TrimSpace(choice)
	for _, pod := range pods {
		if strings.TrimSpace(pod.Display) == trimmed {
			return pod, true
		}
	}
	return PodItem{}, false
}

func splitPodNamespaceArg(value string) (string, string, bool) {
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	ns := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if ns == "" || name == "" {
		return "", "", false
	}
	return ns, name, true
}

func podExistsInNamespace(pods []PodItem, namespace, name string) bool {
	for _, pod := range pods {
		if pod.Name == name && pod.Namespace == namespace {
			return true
		}
	}
	return false
}

func buildPodHeader(context, namespace, selector, podQuery string, allNamespaces bool) string {
	var parts []string
	if context != "" {
		parts = append(parts, "context: "+context)
	}
	if allNamespaces {
		parts = append(parts, "namespace: all")
	} else if namespace != "" {
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

func resolveContext(query string, ignoreFzf bool) (string, error) {
	contexts, err := GetContexts()
	if err != nil {
		return "", err
	}
	if len(contexts) == 0 {
		return "", fmt.Errorf("no kubernetes contexts found")
	}
	if query == "" {
		if ignoreFzf {
			return "", fmt.Errorf("context not specified and fzf is disabled; provide --context <name> or enable fzf")
		}
		if err := checkFzf(); err != nil {
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
	if ignoreFzf {
		return "", fmt.Errorf("context query %q matches multiple entries and fzf is disabled; provide a full context name or enable fzf", query)
	}
	if err := checkFzf(); err != nil {
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

func checkFzf() error {
	if _, err := exec.LookPath("fzf"); err != nil {
		return fmt.Errorf("fzf not found")
	}
	return nil
}

func execOrPrint(context, namespace, pod, container string, command []string, dryRun bool, confirmContext bool, nonInteractive bool) error {
	if dryRun {
		args := ExecArgs(context, namespace, pod, container, command, nonInteractive)
		fmt.Fprintln(os.Stdout, "kubectl "+strings.Join(args, " "))
		return nil
	}
	if confirmContext && confirmContextMatch(context, namespace) {
		if err := confirmContextPrompt(context, namespace); err != nil {
			return err
		}
	}
	return ExecPod(context, namespace, pod, container, command, nonInteractive)
}
