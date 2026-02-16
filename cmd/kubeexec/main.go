package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	var ignoreFzf bool
	var allNamespaces bool
	pflag.BoolVarP(&showVersion, "version", "v", false, "print version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "show this message")
	pflag.StringVar(&context, "context", "", "kubernetes context (overrides current context)")
	if f := pflag.Lookup("context"); f != nil {
		f.NoOptDefVal = ""
	}
	pflag.StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace (defaults to current context/namespace)")
	pflag.StringVarP(&container, "container", "c", "", "container name (defaults to pod's default)")
	pflag.StringVarP(&selector, "selector", "l", "", "label selector for pods (e.g. app=api)")
	pflag.BoolVarP(&allNamespaces, "all-namespaces", "A", false, "list pods across all namespaces")
	pflag.BoolVar(&dryRun, "dry-run", false, "print kubectl command without executing")
	pflag.Var(newConfirmBoolFlag(&confirmContext), "confirm-context", "confirm when context/namespace looks like prod (values: true/True/1/on/ON/false/False/0/off/OFF; env: KUBEEXEC_CONFIRM_CONTEXT; config: ~/.config/kubeexec/kubeexec.toml, TOML boolean)")
	if f := pflag.Lookup("confirm-context"); f != nil {
		f.NoOptDefVal = "true"
	}
	pflag.Var(newConfirmBoolFlag(&nonInteractive), "non-interactive", "run without stdin or TTY (no -i/-t), useful for scripts (values: true/True/1/on/ON/false/False/0/off/OFF; env: KUBEEXEC_NON_INTERACTIVE; config: ~/.config/kubeexec/kubeexec.toml, TOML boolean)")
	if f := pflag.Lookup("non-interactive"); f != nil {
		f.NoOptDefVal = "true"
	}
	pflag.Usage = func() {
		cmd := displayName()
		fmt.Fprintln(os.Stdout, "USAGE:")
		fmt.Fprintf(os.Stdout, "  %s                          : select a pod and exec into it\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s <POD>                    : exec into a specific pod (exact or partial)\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s --context <CTX>          : use a specific kubernetes context (exact or partial)\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s --context                : select a context from a list\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s <POD> -- <CMD> [ARGS]     : run a command in a specific pod\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s -- <CMD> [ARGS]           : select a pod, then run a command\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s <POD> -c <NAME>          : exec into a specific container in a pod\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s -n <NS> -c <NAME>        : specify both namespace and container\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s -n <NS> -l <SEL>         : specify both namespace and selector\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s -A, --all-namespaces     : select a pod across all namespaces\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s -A <NS>/<POD>            : target a pod across all namespaces directly\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s version, -v, --version   : print version and exit\n", cmd)
		fmt.Fprintf(os.Stdout, "  %s -h, --help               : show this message\n", cmd)
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "ALIASES:")
		fmt.Fprintln(os.Stdout, "  kubeexec")
		fmt.Fprintln(os.Stdout, "  kubectl xc")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "OPTIONS:")
		pflag.CommandLine.SetOutput(os.Stdout)
		pflag.CommandLine.PrintDefaults()
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "NOTES:")
		fmt.Fprintln(os.Stdout, "  - A kubectl context must be set unless --context is provided")
		fmt.Fprintln(os.Stdout, "  - Uses the context namespace when -n is not provided")
		fmt.Fprintln(os.Stdout, "  - If --context or <POD> is ambiguous, fzf picker is used")
		fmt.Fprintln(os.Stdout, "  - If fzf is disabled and selection is required, the command exits with an error")
		fmt.Fprintln(os.Stdout, "  - If pod has multiple containers, default is used when available; otherwise picker is shown")
		fmt.Fprintln(os.Stdout, "  - Config precedence: flag > env > ~/.config/kubeexec/kubeexec.toml")
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
	ignoreFzfEnabled, err := cmdutil.ResolveIgnoreFzf(false, ignoreFzf)
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

	if err := cmdutil.Run(cmdutil.RunOptions{
		Context:          context,
		Namespace:        namespace,
		Container:        container,
		Selector:         selector,
		Pod:              pod,
		Command:          commandArgs,
		DryRun:           dryRun,
		ContextRequested: contextRequested,
		ConfirmContext:   confirmContextEnabled,
		NonInteractive:   nonInteractiveEnabled,
		IgnoreFzf:        ignoreFzfEnabled,
		AllNamespaces:    allNamespaces,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func displayName() string {
	name := filepath.Base(os.Args[0])
	const pluginPrefix = "kubectl-"
	if strings.HasPrefix(name, pluginPrefix) {
		return "kubectl " + strings.TrimPrefix(name, pluginPrefix)
	}
	return "kubeexec"
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

type confirmBoolFlag struct {
	value *bool
}

func newConfirmBoolFlag(value *bool) *confirmBoolFlag {
	return &confirmBoolFlag{value: value}
}

func (b *confirmBoolFlag) String() string {
	if b == nil || b.value == nil {
		return "false"
	}
	return strconv.FormatBool(*b.value)
}

func (b *confirmBoolFlag) Set(value string) error {
	parsed, ok := cmdutil.ParseConfirmBool(value)
	if !ok {
		return fmt.Errorf("invalid value %q (use true/True/1/on/ON/false/False/0/off/OFF)", value)
	}
	*b.value = parsed
	return nil
}

func (b *confirmBoolFlag) Type() string {
	return "bool"
}
