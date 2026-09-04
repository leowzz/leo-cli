package shellinit

import (
	"fmt"

	"github.com/leo/leo-cli/internal/version"
)

const posixScriptTemplate = `# leo shell integration
repo() {
	local target
	target="$(%s repo)"
	if [ -n "$target" ]; then
		cd "$target"
	fi
}
`

const zshCompletionTemplate = `
# leo completion
if (( ! $+functions[compdef] )); then
	autoload -Uz compinit
	compinit
fi
eval "$(%s completion zsh)"
`

func Script(shell string) (string, error) {
	switch shell {
	case "zsh":
		commandName := version.CommandName()
		return fmt.Sprintf(posixScriptTemplate, commandName) + fmt.Sprintf(zshCompletionTemplate, commandName), nil
	case "bash":
		return fmt.Sprintf(posixScriptTemplate, version.CommandName()), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}
