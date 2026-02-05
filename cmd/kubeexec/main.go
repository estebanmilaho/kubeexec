package main

import (
	"fmt"
	"os"

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
	pflag.BoolVar(&showVersion, "version", false, "print version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "show this message")
	pflag.StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace (defaults to current context/namespace)")
	pflag.StringVarP(&container, "container", "c", "", "container name (defaults to pod's default)")
	pflag.StringVarP(&selector, "selector", "l", "", "label selector for pods (e.g. app=api)")
	pflag.Usage = func() {
		fmt.Fprint(os.Stdout, `USAGE:
  kubeexec                          : select a pod and exec into it
  kubeexec -n, --namespace <NS>     : use a specific namespace
  kubeexec -c, --container <NAME>   : exec into a specific container
  kubeexec -l, --selector <SEL>     : filter pods by label selector
  kubeexec -n <NS> -c <NAME>        : specify both namespace and container
  kubeexec -n <NS> -l <SEL>         : specify both namespace and selector
  kubeexec -version                 : print version and exit
  kubeexec -h, --help               : show this message

NOTES:
  - A kubectl context has to be set
  - Uses current kubectl namespace when -n is not provided
  - If the pod has multiple containers and no default, you will be prompted with fzf
  - If a default container exists, it will be used; pass -c to override
  - Selector uses standard kubectl label selector syntax (e.g. app=api,env=prod)
`)
	}
	pflag.Parse()

	if showHelp {
		pflag.Usage()
		return
	}

	if showVersion {
		fmt.Println(version)
		return
	}

	if err := cmdutil.Run(namespace, container, selector); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
