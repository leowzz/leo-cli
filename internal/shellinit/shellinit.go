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

func Script(shell string) (string, error) {
	switch shell {
	case "zsh", "bash":
		return fmt.Sprintf(posixScriptTemplate, version.CommandName()), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}
