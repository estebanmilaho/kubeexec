complete -c kubeexec -s h -l help -d "show this message"
complete -c kubeexec -s v -l version -d "print version and exit"
complete -c kubeexec -s n -l namespace -d "kubernetes namespace (defaults to current context/namespace)" -r
complete -c kubeexec -s c -l container -d "container name (defaults to pod's default)" -r
complete -c kubeexec -s l -l selector -d "label selector for pods (e.g. app=api)" -r
complete -c kubeexec -l context -d "kubernetes context (overrides current context)" -r
complete -c kubeexec -l dry-run -d "print the kubectl exec command and exit"
