package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"kubeexec/internal/cmdutil"
)

var version = "dev"

func main() {
	var showVersion bool
	var showHelp bool
	var namespace string
	var container string
	var selector string
	var context string
	var dryRun bool
	var pod string
	var confirmContext bool
	var nonInteractive bool
	pflag.BoolVarP(&showVersion, "version", "v", false, "print version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "show this message")
	pflag.StringVar(&context, "context", "", "kubernetes context (overrides current context)")
	if f := pflag.Lookup("context"); f != nil {
		f.NoOptDefVal = ""
	}
	pflag.StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace (defaults to current context/namespace)")
	pflag.StringVarP(&container, "container", "c", "", "container name (defaults to pod's default)")
	pflag.StringVarP(&selector, "selector", "l", "", "label selector for pods (e.g. app=api)")
	pflag.BoolVar(&dryRun, "dry-run", false, "print kubectl command without executing")
	pflag.BoolVar(&confirmContext, "confirm-context", false, "confirm when context/namespace looks like prod (env: KUBEEXEC_CONFIRM_CONTEXT, config: ~/.config/kubeexec)")
	pflag.BoolVar(&nonInteractive, "non-interactive", false, "run without stdin or TTY (no -i/-t), useful for scripts")
	pflag.Usage = func() {
		fmt.Fprint(os.Stdout, `USAGE:
  kubeexec                          		: select a pod and exec into it
  kubeexec <POD>                    		: exec into a specific pod (exact or partial)
  kubeexec --context <CTX>          		: use a specific kubernetes context (exact or partial)
  kubeexec --context                		: select a context from a list
  kubeexec -n, --namespace <NS>     		: use a specific namespace
  kubeexec -c, --container <NAME>   		: exec into a specific container
  kubeexec -l, --selector <SEL>     		: filter pods by label selector
  kubeexec --dry-run                		: print the kubectl exec command and exit
  kubeexec --non-interactive[=true|false]   : run without stdin or TTY (no -i/-t)
  kubeexec --confirm-context[=true|false] 	: confirm when context/namespace looks like prod
  kubeexec <POD> -c <NAME>          		: exec into a specific container in a pod
  kubeexec -n <NS> -c <NAME>        		: specify both namespace and container
  kubeexec -n <NS> -l <SEL>         		: specify both namespace and selector
  kubeexec version, -v, --version   		: print version and exit
  kubeexec -h, --help               		: show this message

NOTES:
  - A kubectl context must be set unless --context is provided
  - Uses the context namespace when -n is not provided
  - If the pod has multiple containers and no default, you will be prompted with fzf
  - If a default container exists, it will be used; pass -c to override
  - Selector uses standard kubectl label selector syntax (e.g. app=api,env=prod)
  - If --context or <POD> matches multiple entries, you will be prompted with fzf
  - Confirm context can be configured via --confirm-context, KUBEEXEC_CONFIRM_CONTEXT, or ~/.config/kubeexec (true/True/1/false/False/0)
`)
	}
	pflag.CommandLine.SetOutput(io.Discard)
	if err := rejectDeprecatedArgs(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr)
		pflag.Usage()
		os.Exit(2)
	}
	if err := pflag.CommandLine.Parse(normalizeContextArgs(os.Args[1:])); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr)
		pflag.Usage()
		os.Exit(2)
	}
	contextRequested := false
	if f := pflag.Lookup("context"); f != nil && f.Changed {
		contextRequested = true
	}
	confirmContextRequested := false
	if f := pflag.Lookup("confirm-context"); f != nil && f.Changed {
		confirmContextRequested = true
	}
	confirmContextEnabled, err := cmdutil.ResolveConfirmContext(confirmContextRequested, confirmContext)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	args := pflag.Args()
	if len(args) > 0 && args[0] == "version" {
		fmt.Println(version)
		return
	}
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: too many arguments")
		pflag.Usage()
		os.Exit(2)
	}
	if len(args) == 1 {
		pod = args[0]
	}

	if showHelp {
		pflag.Usage()
		return
	}

	if showVersion {
		fmt.Println(version)
		return
	}

	if err := cmdutil.Run(context, namespace, container, selector, pod, dryRun, contextRequested, confirmContextEnabled, nonInteractive); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func normalizeContextArgs(args []string) []string {
	normalized := make([]string, 0, len(args))
	flagsAllowEmpty := map[string]struct{}{
		"--context":   {},
		"--container": {},
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if _, ok := flagsAllowEmpty[arg]; ok {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				normalized = append(normalized, arg+"=")
				continue
			}
		}
		normalized = append(normalized, arg)
	}
	return normalized
}

func rejectDeprecatedArgs(args []string) error {
	for _, arg := range args {
		if arg == "-version" {
			return fmt.Errorf("unknown flag: -version (use -v or --version)")
		}
	}
	return nil
}
