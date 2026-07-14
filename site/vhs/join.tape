Output site/public/demos/join.tmp.webm
Require bin/leo
Set Shell "bash"
Set Width 1000
Set Height 600
Set FontSize 20
Set TypingSpeed 45ms
Set Theme { "name": "leo", "black": "#161616", "red": "#ef4444", "green": "#22c55e", "yellow": "#eab308", "blue": "#3b82f6", "magenta": "#d946ef", "cyan": "#14b8a6", "white": "#f5f5f5", "brightBlack": "#737373", "brightRed": "#f87171", "brightGreen": "#4ade80", "brightYellow": "#facc15", "brightBlue": "#60a5fa", "brightMagenta": "#e879f9", "brightCyan": "#2dd4bf", "brightWhite": "#ffffff", "background": "#161616", "foreground": "#f5f5f5", "selection": "#404040", "cursor": "#f5f5f5" }

Type "bin/leo join site/vhs/fixtures/users.csv"
Sleep 0.5s
Enter
Wait+Screen /user_id in/
Sleep 0.8s
Right
Sleep 0.8s
Down 2
Sleep 0.8s
Type "u"
Sleep 3s
Hide
Escape
