_kubeexec() {
	local cur prev
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[COMP_CWORD-1]}"

	case "$prev" in
		-n|--namespace|-c|--container|-l|--selector|--context)
			return 0
			;;
	esac

	local flags="--context --namespace --container --selector --dry-run --version --help -n -c -l -h"
	COMPREPLY=($(compgen -W "${flags}" -- "$cur"))
}

complete -F _kubeexec kubeexec
