package cmdutil

import (
    "fmt"
    "os/exec"
)

func Run() error {
    if _, err := exec.LookPath("kubectl"); err != nil {
        return fmt.Errorf("kubectl not found")
    }
    if _, err := exec.LookPath("fzf"); err != nil {
        return fmt.Errorf("fzf not found")
    }

    context, err := CurrentContext()
    if err != nil {
        return err
    }
    if context == "" {
        return fmt.Errorf("no kubernetes context is set")
    }

    namespace, err := CurrentNamespace()
    if err != nil {
        return err
    }
    if namespace == "" {
        namespace = "default"
    }

    pods, err := GetPods(namespace)
    if err != nil {
        return err
    }
    if len(pods) == 0 {
        return fmt.Errorf("no pods found")
    }

    choice, err := ChooseWithFzf(pods)
    if err != nil {
        return err
    }
    if choice == "" {
        return fmt.Errorf("no pod selected")
    }

    return ExecPod(choice)
}
