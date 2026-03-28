package cli

import (
	"fmt"
	"io"
)

func cmdCompletion(args []string, stdout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "Usage: bujotui completion [bash|zsh]")
		return 1
	}

	switch args[0] {
	case "bash":
		fmt.Fprint(stdout, bashCompletion)
		return 0
	case "zsh":
		fmt.Fprint(stdout, zshCompletion)
		return 0
	default:
		fmt.Fprintf(stdout, "Unknown shell: %s (supported: bash, zsh)\n", args[0])
		return 1
	}
}

const bashCompletion = `# bujotui bash completion
# Add to ~/.bashrc:  eval "$(bujotui completion bash)"

_bujotui() {
    local cur prev commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    commands="add list done migrate schedule cancel remove projects people config version help completion"

    case "${prev}" in
        bujotui)
            COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
            return 0
            ;;
        config)
            COMPREPLY=( $(compgen -W "init" -- "${cur}") )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh" -- "${cur}") )
            return 0
            ;;
        -s|--symbol)
            return 0
            ;;
        -p|--project)
            return 0
            ;;
        -a|--person)
            return 0
            ;;
        --dir)
            COMPREPLY=( $(compgen -d -- "${cur}") )
            return 0
            ;;
    esac

    case "${COMP_WORDS[1]}" in
        add)
            COMPREPLY=( $(compgen -W "-s -p -a -d" -- "${cur}") )
            return 0
            ;;
        list)
            COMPREPLY=( $(compgen -W "--project --person --symbol --date --week --month --time" -- "${cur}") )
            return 0
            ;;
    esac

    return 0
}

complete -F _bujotui bujotui
`

const zshCompletion = `# bujotui zsh completion
# Add to ~/.zshrc:  eval "$(bujotui completion zsh)"

_bujotui() {
    local -a commands
    commands=(
        'add:Add a new entry'
        'list:List entries'
        'done:Mark entry as done'
        'migrate:Mark entry as migrated'
        'schedule:Mark entry as scheduled'
        'cancel:Mark entry as cancelled'
        'remove:Remove an entry'
        'projects:List known projects'
        'people:List known people'
        'config:Show or initialize configuration'
        'version:Show version'
        'help:Show help'
        'completion:Output shell completion script'
    )

    _arguments -C \
        '--dir[Override config and data directory]:directory:_files -/' \
        '1:command:->command' \
        '*::arg:->args'

    case "$state" in
        command)
            _describe 'command' commands
            ;;
        args)
            case "${words[1]}" in
                add)
                    _arguments \
                        '-s[Symbol name]:symbol:' \
                        '-p[Project name]:project:' \
                        '-a[Person name]:person:' \
                        '-d[Date/time]:datetime:' \
                        '*:description:'
                    ;;
                list)
                    _arguments \
                        '--project[Filter by project]:project:' \
                        '--person[Filter by person]:person:' \
                        '--symbol[Filter by symbol]:symbol:' \
                        '--date[Specific date]:date:' \
                        '--week[Current week]' \
                        '--month[Current month]' \
                        '--time[Show timestamps]'
                    ;;
                config)
                    _arguments '1:subcommand:(init)'
                    ;;
                completion)
                    _arguments '1:shell:(bash zsh)'
                    ;;
                done|migrate|schedule|cancel|remove)
                    _arguments '1:entry number:'
                    ;;
            esac
            ;;
    esac
}

compdef _bujotui bujotui
`
