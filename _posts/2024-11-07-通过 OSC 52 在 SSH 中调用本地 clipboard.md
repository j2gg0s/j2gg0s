我日常使用 MacbookPro, 但有一半的开发工作通过 SSH 在 Linux/amd64 的环境下完成.
考虑到我基本使用 vim 做为编辑器, 上述方案的唯一痛点在于两台机器无法共享剪贴盘.

昨天乘着配置新环境的机会, 搜索了一下解决方案, 选择了 OSC 52.

OSC 是 Operating System Command 的缩写, 其约定了一系列特殊符号的含义.
[OSC 52](https://www.reddit.com/r/vim/comments/k1ydpn/a_guide_on_how_to_copy_text_from_anywhere/)
是其一个子类, 用于控制系统剪贴板. 我们引用 tmux 的文档来总结其原理:

> Some terminals offer an escape sequence to set the clipboard.
> This is one of the operating system control sequences so it is known as OSC 52.
>
> The way it works is that when text is copied in tmux it is packaged up and sent to the outside terminal in a similar way to how tmux draws the text and colours and attributes. The outside terminal recognises the clipboard escape sequence and sets the system clipboard.

当前大部分的终端都已经支持了这项功能,
当你在终端执行 `echo -en "\e]52;c;$(base64 <<< OSC)\a"` 后再黏贴可以看到 OSC 这三个字符串.

OSC 52 在 tmux 中的定义是 `\033]52;%p1%s;%p2%s\a`, 其中各项的含义是:
- `\033` 是一个八进制表示的 27, 对应控制符号 ESC. 我们也可以使用 `\x1B`(十六进制) 或 `\e`(简写). 
- `]52` 代表 OSC 52
- `%p1%s` 是 tmux 的占位符, 实际使用时 `c` 代表剪贴板
- `%p2%s` 对应 base64 后的剪贴内容.
- `\a` 是控制字符 BEL 的缩写, 也可以使用 `\x7` 或 `\07`.

我在 iterm2 中使用 tmux, 参考 [tmux 的官方文档](https://github.com/tmux/tmux/wiki/Clipboard), 可以知道关键的几项设置:
1. iterm2 需要允许内部应用访问系统剪贴板, [Link](https://github.com/tmux/tmux/wiki/Clipboard#terminal-support---iterm2)
2. tmux 配置 TERM, 检查方式为 `tmux info | grep Ms`, 设置方式为 `set-option -as terminal-overrides ",xterm-256color:clipboard"`
3. tmux 配置 set-clipboard, 设置访问是 `set -g set-clipboard on`
4. 通过 `tmux source-file` 更新配置

需要额外谈的是 set-clipboard, 文档推荐的设置是 external, 但我选择的是 on.
二者的区别在于 on 允许 tmux 和 tmux 内的应用设置剪贴板, external 仅允许 tmux 设置剪贴板.
所以当我通过 ssh 访问开发机并希望在开发机内访问本机的剪贴板时需要将 set-clipboard 设置为 on.

最后通过 [remote-pbcopy-iterm2](https://github.com/skaji/remote-pbcopy-iterm2/tree/master) 将 OSC 52 的逻辑封装成了命令 pbcopy.
