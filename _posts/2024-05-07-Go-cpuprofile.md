The Go runtime provides [profiling data](https://pkg.go.dev/runtime/pprof)
in the format expected by the
[pprof visualization tool](https://github.com/google/pprof/blob/main/doc/README.md).

从这句话里可以直观的看出两点:
- 利用了已有的可视化工具 [google/pprof](https://github.com/google/pprof)
- Go runtime 负责生成其所需的数据

在 CPU Profiling 中, 这些数据是通过定时采样生成的.
在 [SetCPUProfileRate](https://github.com/golang/go/blob/master/src/runtime/cpuprof.go#L68) 中:
- 利用 [setitimer](https://linux.die.net/man/2/setitimer) 实现每 10ms 发送一次 SIGPROF
- 同时注册 SIGPROF 的处理函数为 [sigprof](https://github.com/golang/go/blob/master/src/runtime/proc.go#L5285)

采样的核心逻辑是展开被中断的 goroutine 的调用栈, 将其写入一个 lock-free buffer.
随后由另一个 goroutine 从 buffer 中读取调用栈, 添加辅助信息后写入指定文件.
