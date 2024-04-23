Go 有一套非常好用的 pprof, 我非常高兴能有时间和兴趣可以去了解其细节.

从 [net/http/pprof](https://github.com/golang/go/blob/go1.21.9/src/net/http/pprof/pprof.go#L133)
中我们可以验证 go 中 cpuprofile 的入口是
[pprof.StartCPUProfile](https://github.com/golang/go/blob/master/src/runtime/pprof/pprof.go#L812).
```go
// Profile responds with the pprof-formatted cpu profile.
// Profiling lasts for duration specified in seconds GET parameter, or for 30 seconds if not specified.
// The package initialization registers it as /debug/pprof/profile.
func Profile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	sec, err := strconv.ParseInt(r.FormValue("seconds"), 10, 64)
	if sec <= 0 || err != nil {
		sec = 30
	}

	if durationExceedsWriteTimeout(r, float64(sec)) {
		serveError(w, http.StatusBadRequest, "profile duration exceeds server's WriteTimeout")
		return
	}

	// Set Content Type assuming StartCPUProfile will work,
	// because if it does it starts writing.
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="profile"`)
	if err := pprof.StartCPUProfile(w); err != nil {
		// StartCPUProfile failed, so no writes yet.
		serveError(w, http.StatusInternalServerError,
			fmt.Sprintf("Could not enable CPU profiling: %s", err))
		return
	}
	sleep(r, time.Duration(sec)*time.Second)
	pprof.StopCPUProfile()
}
```

StartCPUProfile 的核心逻辑包括:
- 注册 SIGPROF 的处理函数, 使得 SIGPROF 发生时调用 [sigprof](https://github.com/golang/go/blob/master/src/runtime/proc.go#L5285)
- 通过 [setitimer](https://linux.die.net/man/2/setitimer) 10ms 发送一次 SIGPROF

[sigprof]() 会采集中断前的调用栈, 按照 [google/pprof](https://github.com/google/pprof) 定义的格式保存在文件中.
后续我们就可以使用 [google/pprof]() 进行分析和可视化.
[go tool pprof](https://github.com/golang/go/tree/master/src/cmd/pprof) 基本来源于 [google/pprof]().

对于使用来说, 仔细并完整的阅读 [Profiling Go Programs](https://go.dev/blog/pprof) 依然是最佳的选择.
但当你注意到 CPU Profile 来源于对调用栈的周期性采样时, 你应该更容易理解 flat 和 cum 的区别.
前者时采样时正在被执行的次数, 后者时采样时出现在调用栈上的次数.
