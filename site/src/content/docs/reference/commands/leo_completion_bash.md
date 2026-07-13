---
title: "leo completion bash"
description: "由 leo CLI 命令树生成的命令参考。"
---

> 本页由 Cobra 命令树生成；命令、参数和输出保持 CLI 原文。

## leo completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(leo completion bash)

To load completions for every new session, execute once:

#### Linux:

	leo completion bash > /etc/bash_completion.d/leo

#### macOS:

	leo completion bash > $(brew --prefix)/etc/bash_completion.d/leo

You will need to start a new shell for this setup to take effect.


```
leo completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -v, --version   Print version
```

### SEE ALSO

* [leo completion](../leo_completion/)	 - Generate the autocompletion script for the specified shell

