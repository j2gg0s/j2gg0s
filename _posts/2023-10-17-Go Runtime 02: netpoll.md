最近重看 Go 的 runtime 的时候发现了
[netpoll.go](https://github.com/golang/go/blob/go1.21.1/src/runtime/netpoll.go).
以为时最近加入的内容, 但 git blame 下却发现时很早以前就存在了.
非常佩服自己的主动过滤系统.

netpoll 是 Go 是对不同平台 IO 的封装, 比如 Linux's 的 epoll.
一方面是和其他语言的三方库一样, 简化上游使用者的心智负担.
另一方, 通过实现在 runtime, 和 GMP 调度模型高度结合, 尽可能的提高效率.

## IO 相关的基础知识
我们首先明确两组属性.

阻塞和非阻塞 IO, 前者指调用阻塞在套接字上直到有数据可以读取/写入,
后者指在没有数据可以读取/无法写入数据时调用直接返回, 需要由调用者采用轮训等方法决定处理方式.

同步和异步 IO, 前者指调用直接生效, 执行完后返回结果,
后者指调用立刻返回, 但具体工作后续执行, 结果通过某种方式通知调用者.

Linux 的套接字(socket) 即支持阻塞模式也支持非阻塞模式, 可以在创建时通过参数指定.
但一般是同步的, 异步 socket 似乎在 windows 中比较流行.

而诸如 Nginx 这类网关需要同时处理成千上万的链接, 创建对等数量的线程阻塞在链接上是不现实的,
他们一般会选择同步非阻塞 IO, 配合轮训来让一个线程处理成百上千个链接.

但 IO 操作一般是系统调用，涉及内核态到系统态的转换, 异常昂贵.
所以我们通过多路复用, 实现一次调用操作多个套接字来减少对系统 API 的调用.
这在 Linux 上对应 select/poll/epoll, 前两者需要在轮训是传入所有关注的套接字, 导致高昂的内存成本.
epoll 通过拆分处单独的注册接口, 使得高频的轮训操作不需要传入所有套接字, 进而取得更高的效率,
成为 Linux 下的主流选择.

## 如何使用 netpoll
Go 中的 TCP 使用 netpoll 实现, 我们以 Read 操作为例来观察如何使用 netpoll.

首先 Read 操作会被委托给 `internal/poll.FD`:
```go
// TCPConn is an implementation of the Conn interface for TCP network
// connections.
type TCPConn struct {
	conn
}
...
type conn struct {
	fd *netFD
}
...
// Network file descriptor.
type netFD struct {
	pfd poll.FD

	// immutable until Close
	family      int
	sotype      int
	isConnected bool // handshake completed or use of association with peer
	net         string
	laddr       Addr
	raddr       Addr
}
...
func (fd *netFD) Read(p []byte) (n int, err error) {
	n, err = fd.pfd.Read(p)
	runtime.KeepAlive(fd)
	return n, wrapSyscallError(readSyscallName, err)
}
```

`internal/poll.FD` 是对 `runtime/netpoll.go` 的抽象和封装,
`FD.Read` 内部首先尝试直接读取数据, 如果没有则阻塞在套接字上, 等待数据到达后触发下一次尝试.
阻塞这个操作通过 `runtime.pollWait` 实现.
```go
// Read implements io.Reader.
func (fd *FD) Read(p []byte) (int, error) {
    ...
	for {
		n, err := ignoringEINTRIO(syscall.Read, fd.Sysfd, p)
		if err != nil {
			n = 0
			if err == syscall.EAGAIN && fd.pd.pollable() {
				if err = fd.pd.waitRead(fd.isFile); err == nil {
					continue
				}
			}
		}
		err = fd.eofError(n, err)
		return n, err
	}
}
...
func (pd *pollDesc) wait(mode int, isFile bool) error {
	if pd.runtimeCtx == 0 {
		return errors.New("waiting for unsupported file type")
	}
	res := runtime_pollWait(pd.runtimeCtx, mode)
	return convertErr(res, isFile)
}

func (pd *pollDesc) waitRead(isFile bool) error {
	return pd.wait('r', isFile)
}
```

Read 的前提条件是通过 pd.init 将套接字注册到 netpoll.
```go
func (fd *netFD) accept() (netfd *netFD, err error) {
	d, rsa, errcall, err := fd.pfd.Accept()
	if err != nil {
		if errcall != "" {
			err = wrapSyscallError(errcall, err)
		}
		return nil, err
	}

	if netfd, err = newFD(d, fd.family, fd.sotype, fd.net); err != nil {
		poll.CloseFunc(d)
		return nil, err
	}
	if err = netfd.init(); err != nil {
		netfd.Close()
		return nil, err
	}
	lsa, _ := syscall.Getsockname(netfd.pfd.Sysfd)
	netfd.setAddr(netfd.addrFunc()(lsa), netfd.addrFunc()(rsa))
	return netfd, nil
}
...
// Init initializes the FD. The Sysfd field should already be set.
// This can be called multiple times on a single FD.
// The net argument is a network name from the net package (e.g., "tcp"),
// or "file".
// Set pollable to true if fd should be managed by runtime netpoll.
func (fd *FD) Init(net string, pollable bool) error {
	fd.SysFile.init()

	// We don't actually care about the various network types.
	if net == "file" {
		fd.isFile = true
	}
	if !pollable {
		fd.isBlocking = 1
		return nil
	}
	err := fd.pd.init(fd)
	if err != nil {
		// If we could not initialize the runtime poller,
		// assume we are using blocking mode.
		fd.isBlocking = 1
	}
	return err
}
```

## netpoll 的实现
在上一节中, 我们通过 Read 已经明确 netpoll 的使用方式:
- 通过 runtime.pollOpen 将套接字注册到 netpoll
- 通过 runtime.pollWait 让出 CPU 的使用权直到数据到达

这些函数都在 [runtime/netpoll.go](https://github.com/golang/go/blob/master/src/runtime/netpoll.go), 包含了 netpoll 的逻辑代码.
[runtime/netpoll_epoll.go](https://github.com/golang/go/blob/master/src/runtime/netpoll_epoll.go) 则是对 epoll 相关的系统调用的简单封装.

netpoll 的核心逻辑在 pollDesc 的状态字段 rg&wg.
字段可能有四种值:
- pdReady, 代表此时句柄可读
- pdWait, 即将有 goroutine 阻塞在句柄上
- 指向 goroutine 的指针, 代表阻塞在句柄上的 goroutine
- pdNil, 即没有阻塞的 goroutine, 也没有数据可读

此处主要的奥秘在于:
- 向 epoll 注册句柄时传入的 epoll_event, 除了可以传递关注的事件, 还可以保存自定义的事件, 此处为指向 pollDesc 的指针
- goroutine 阻塞在句柄上时, 会将 pollDesc 的 rg/wg 修改指向自身的指针
- 所以当某个句柄就绪时, 可以找到对应的 pollDesc, 并判断是否有 goroutine 阻塞与此句柄

在这个基础上, Go 的调度器在调度时会查看是否有之前阻塞在 epoll 上当现在就绪的 goroutine, 如果存在则将其加入调度中.
