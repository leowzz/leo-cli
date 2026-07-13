---
title: "leo completion powershell"
description: "由 leo CLI 命令树生成的命令参考。"
---

> 本页由 Cobra 命令树生成；命令、参数和输出保持 CLI 原文。

## leo completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	leo completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
leo completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -v, --version   Print version
```

### SEE ALSO

* [leo completion](../leo_completion/)	 - Generate the autocompletion script for the specified shell

