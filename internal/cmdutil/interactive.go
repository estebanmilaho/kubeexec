package cmdutil

import (
    "bufio"
    "fmt"
    "io"
    "os"
    "os/exec"
    "strings"
)

func ChooseWithFzf(items []string) (string, error) {
    cmd := exec.Command("fzf", "--ansi", "--no-preview")
    in, err := cmd.StdinPipe()
    if err != nil {
        return "", err
    }
    out, err := cmd.StdoutPipe()
    if err != nil {
        return "", err
    }
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        return "", err
    }

    go func() {
        w := bufio.NewWriter(in)
        for _, it := range items {
            fmt.Fprintln(w, it)
        }
        w.Flush()
        in.Close()
    }()

    choiceBytes, err := io.ReadAll(out)
    if err != nil {
        return "", err
    }

    if err := cmd.Wait(); err != nil {
        // fzf returns non-zero on cancel
        return "", nil
    }

    return strings.TrimSpace(string(choiceBytes)), nil
}
