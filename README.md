# kubeexec

Fast `kubectl exec` with fuzzy pod selection.

kubeexec assumes your **context and namespace are already set**, and focuses on fast pod/container selection with safe defaults for interactive shell access.

> [!IMPORTANT]
> Keep `fzf` installed and enabled. kubeexec relies on it for interactive selection when a pod, context, or container is ambiguous. It will run without `fzf`, but selection features are limited and may fail fast when a choice is required.


**Workflow companion:** [kubectx/kubens](https://github.com/ahmetb/kubectx) for fast context and namespace switching.

## Requirements
- `kubectl`
- `fzf` (recommended)

## Installation

### Homebrew (macOS/Linux)
```bash
brew tap estebanmilaho/kubeexec
brew install kubeexec
```

### From GitHub releases
Download the appropriate archive from the releases page and place `kubeexec` in your `PATH`.

### From source
```bash
go build -o kubeexec ./cmd/kubeexec
sudo mv kubeexec /usr/local/bin/
```

## Usage
```bash
kubeexec
kubeexec <POD>
kubeexec --context <CTX>
kubeexec --context
kubeexec <POD> -c <NAME>
kubeexec -n <NS> -l <SEL>
kubeexec -- <CMD> [ARGS]
kubeexec <POD> -- <CMD> [ARGS]
```

## Examples
```bash
# Select a pod from the current namespace and exec into it
kubeexec

# Run a command in a specific pod
kubeexec app-123 -- ls -la /var/log

# Use a specific namespace and label selector
kubeexec -n kube-system -l k8s-app=kube-dns

# Non-interactive execution (no -i/-t)
kubeexec --non-interactive app-123 -- cat /etc/os-release
```

## Behavior
- Uses the current context/namespace by default.
- If multiple pods match and `fzf` is enabled, you will be prompted to choose.
- If the pod has multiple containers and no default, you will be prompted to choose.
- `--` passes a command directly to `kubectl exec` instead of starting a shell.

## Configuration
Config file path (including Homebrew installs):
```
~/.config/kubeexec/kubeexec.toml
```

TOML booleans only:
```toml
confirm-context = true
non-interactive = false
ignore-fzf = false
```

Environment variables:
- `KUBEEXEC_CONFIRM_CONTEXT`
- `KUBEEXEC_NON_INTERACTIVE`
- `KUBEEXEC_IGNORE_FZF`

Values accepted for env and flags: `true`, `True`, `1`, `on`, `ON`, `false`, `False`, `0`, `off`, `OFF`.

## Notes on fzf
- If `fzf` is disabled and a selection is required, kubeexec fails fast with a clear error.
- Default is `fzf` enabled.

## Version
```bash
kubeexec --version
kubeexec version
```
