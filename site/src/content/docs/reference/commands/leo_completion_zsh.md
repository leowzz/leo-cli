---
title: "leo completion zsh"
description: "由 leo CLI 命令树生成的命令参考。"
---

> 本页由 Cobra 命令树生成；命令、参数和输出保持 CLI 原文。

## leo completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(leo completion zsh)

To load completions for every new session, execute once:

#### Linux:

	leo completion zsh > "${fpath[1]}/_leo"

#### macOS:

	leo completion zsh > $(brew --prefix)/share/zsh/site-functions/_leo

You will need to start a new shell for this setup to take effect.


```
leo completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -v, --version   Print version
```

### SEE ALSO

* [leo completion](../leo_completion/)	 - Generate the autocompletion script for the specified shell

