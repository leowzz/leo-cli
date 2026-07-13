---
title: "leo completion fish"
description: "由 leo CLI 命令树生成的命令参考。"
---

> 本页由 Cobra 命令树生成；命令、参数和输出保持 CLI 原文。

## leo completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	leo completion fish | source

To load completions for every new session, execute once:

	leo completion fish > ~/.config/fish/completions/leo.fish

You will need to start a new shell for this setup to take effect.


```
leo completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -v, --version   Print version
```

### SEE ALSO

* [leo completion](./leo_completion.md)	 - Generate the autocompletion script for the specified shell

