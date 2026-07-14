Output site/public/demos/repo-picker.tmp.webm
Require bin/leo
Require site/vhs/setup-repo-demo.sh
Set Shell "bash"
Set Width 1000
Set Height 600
Set FontSize 20
Set TypingSpeed 45ms
Set Theme { "name": "leo", "black": "#161616", "red": "#ef4444", "green": "#22c55e", "yellow": "#eab308", "blue": "#3b82f6", "magenta": "#d946ef", "cyan": "#14b8a6", "white": "#f5f5f5", "brightBlack": "#737373", "brightRed": "#f87171", "brightGreen": "#4ade80", "brightYellow": "#facc15", "brightBlue": "#60a5fa", "brightMagenta": "#e879f9", "brightCyan": "#2dd4bf", "brightWhite": "#ffffff", "background": "#161616", "foreground": "#f5f5f5", "selection": "#404040", "cursor": "#f5f5f5" }
Env XDG_CONFIG_HOME "/tmp/leo-cli-vhs/config"
Env XDG_DATA_HOME "/tmp/leo-cli-vhs/data"

Hide
Type "site/vhs/setup-repo-demo.sh"
Enter
Wait+Screen /ready/
Show
Type "bin/leo repo"
Enter
Wait+Screen /Repositories/
Sleep 1.2s
Type "/"
Sleep 0.2s
Type "bea"
Sleep 2s
Down
Enter
Wait+Screen /beacon/
Sleep 3s
