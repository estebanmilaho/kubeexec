package main

import (
    "flag"
    "fmt"
    "os"

    "kubeexec/internal/cmdutil"
)

var version = "dev"

func main() {
    var showVersion bool
    flag.BoolVar(&showVersion, "version", false, "print version and exit")
    flag.Parse()

    if showVersion {
        fmt.Println(version)
        return
    }

    if err := cmdutil.Run(); err != nil {
        fmt.Fprintln(os.Stderr, "error:", err)
        os.Exit(1)
    }
}
