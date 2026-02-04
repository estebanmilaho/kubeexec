# kubeexec

Select a pod with **fzf** and `kubectl exec` into it.

## Install (local)
```bash
go build -o kubeexec ./cmd/kubeexec
sudo mv kubeexec /usr/local/bin/
```

## Usage
```bash
kubeexec
kubeexec -version
```
