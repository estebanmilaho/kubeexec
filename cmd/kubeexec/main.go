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
	pflag.BoolVar(&nonInteractive, "non-interactive", false, "run without stdin or TTY (no -i/-t), useful for scripts (env: KUBEEXEC_NON_INTERACTIVE, config: ~/.config/kubeexec)")
	pflag.Usage = func() {
		fmt.Fprintln(os.Stdout, "USAGE:")
		fmt.Fprintln(os.Stdout, "  kubeexec                          : select a pod and exec into it")
		fmt.Fprintln(os.Stdout, "  kubeexec <POD>                    : exec into a specific pod (exact or partial)")
		fmt.Fprintln(os.Stdout, "  kubeexec --context <CTX>          : use a specific kubernetes context (exact or partial)")
		fmt.Fprintln(os.Stdout, "  kubeexec --context                : select a context from a list")
		fmt.Fprintln(os.Stdout, "  kubeexec <POD> -- <CMD> [ARGS]     : run a command in a specific pod")
		fmt.Fprintln(os.Stdout, "  kubeexec -- <CMD> [ARGS]           : select a pod, then run a command")
		fmt.Fprintln(os.Stdout, "  kubeexec <POD> -c <NAME>          : exec into a specific container in a pod")
		fmt.Fprintln(os.Stdout, "  kubeexec -n <NS> -c <NAME>        : specify both namespace and container")
		fmt.Fprintln(os.Stdout, "  kubeexec -n <NS> -l <SEL>         : specify both namespace and selector")
		fmt.Fprintln(os.Stdout, "  kubeexec version, -v, --version   : print version and exit")
		fmt.Fprintln(os.Stdout, "  kubeexec -h, --help               : show this message")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "OPTIONS:")
		pflag.CommandLine.SetOutput(os.Stdout)
		pflag.CommandLine.PrintDefaults()
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "NOTES:")
		fmt.Fprintln(os.Stdout, "  - A kubectl context must be set unless --context is provided")
		fmt.Fprintln(os.Stdout, "  - Uses the context namespace when -n is not provided")
		fmt.Fprintln(os.Stdout, "  - If the pod has multiple containers and no default, you will be prompted with fzf")
		fmt.Fprintln(os.Stdout, "  - If a default container exists, it will be used; pass -c to override")
		fmt.Fprintln(os.Stdout, "  - Selector uses standard kubectl label selector syntax (e.g. app=api,env=prod)")
		fmt.Fprintln(os.Stdout, "  - If --context or <POD> matches multiple entries, you will be prompted with fzf")
		fmt.Fprintln(os.Stdout, "  - Confirm context can be configured via --confirm-context, KUBEEXEC_CONFIRM_CONTEXT, or ~/.config/kubeexec (true/True/1/on/ON/false/False/0/off/OFF)")
		fmt.Fprintln(os.Stdout, "  - Non-interactive can be configured via --non-interactive, KUBEEXEC_NON_INTERACTIVE, or ~/.config/kubeexec")
		fmt.Fprintln(os.Stdout, "  - Config file uses key=value lines (confirm-context, non-interactive)")
	}
	pflag.CommandLine.SetOutput(io.Discard)
	flagArgs, commandArgs := splitCommandArgs(os.Args[1:])
	if err := rejectDeprecatedArgs(flagArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr)
		pflag.Usage()
		os.Exit(2)
	}
	if err := pflag.CommandLine.Parse(normalizeContextArgs(flagArgs)); err != nil {
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
	nonInteractiveRequested := false
	if f := pflag.Lookup("non-interactive"); f != nil && f.Changed {
		nonInteractiveRequested = true
	}
	nonInteractiveEnabled, err := cmdutil.ResolveNonInteractive(nonInteractiveRequested, nonInteractive)
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

	if err := cmdutil.Run(context, namespace, container, selector, pod, commandArgs, dryRun, contextRequested, confirmContextEnabled, nonInteractiveEnabled); err != nil {
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

func splitCommandArgs(args []string) (flags []string, command []string) {
	for i, arg := range args {
		if arg == "--" {
			return append([]string(nil), args[:i]...), append([]string(nil), args[i+1:]...)
		}
	}
	return append([]string(nil), args...), nil
}
