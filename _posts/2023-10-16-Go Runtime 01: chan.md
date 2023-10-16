传统的线程模型通过共享内存来实现多个线程之间的相互交流.
依赖精妙的设计和锁等机制来保证程序的正确性.
而 Go 通过 chan 推广了另外一种模式, 即
> Do not communicate by sharing memory; instead, share memory by communicating.

通过汇编代码, 我们可以快速的定位到 chan 相关的函数:
- `runtime.makechan` 用于创建 chan
- `runtime.chansend1` 用于向 chan 发送数据
- `runtime.chanrecv1` 用于从 chan 接收数据
```
138025  func main() {
138026    457680:       49 3b 66 10             cmp    0x10(%r14),%rsp
138027    457684:       0f 86 c1 00 00 00       jbe    45774b <main.main+0xcb>
138028    45768a:       55                      push   %rbp
138029    45768b:       48 89 e5                mov    %rsp,%rbp
138030    45768e:       48 83 ec 40             sub    $0x40,%rsp
138031    457692:       66 44 0f d6 7c 24 38    movq   %xmm15,0x38(%rsp)
138032    457699:       c6 44 24 17 00          movb   $0x0,0x17(%rsp)
138033          ch := make(chan int, 2)
138034    45769e:       48 8d 05 5b 56 00 00    lea    0x565b(%rip),%rax        # 45cd00 <type:*+0x4d00>
138035    4576a5:       bb 02 00 00 00          mov    $0x2,%ebx
138036    4576aa:       e8 71 bd fa ff          call   403420 <runtime.makechan>
138037    4576af:       48 89 44 24 20          mov    %rax,0x20(%rsp)
138038          defer close(ch)
138039    4576b4:       44 0f 11 7c 24 28       movups %xmm15,0x28(%rsp)
138040    4576ba:       48 8d 0d 9f 00 00 00    lea    0x9f(%rip),%rcx        # 457760 <main.main.func1>
138041    4576c1:       48 89 4c 24 28          mov    %rcx,0x28(%rsp)
138042    4576c6:       48 89 44 24 30          mov    %rax,0x30(%rsp)
138043    4576cb:       48 8d 4c 24 28          lea    0x28(%rsp),%rcx
138044    4576d0:       48 89 4c 24 38          mov    %rcx,0x38(%rsp)
138045    4576d5:       c6 44 24 17 01          movb   $0x1,0x17(%rsp)
138046          ch <- 10
138047    4576da:       48 8d 1d f7 68 02 00    lea    0x268f7(%rip),%rbx        # 47dfd8 <runtime.egcbss+0x5>
138048    4576e1:       e8 1a bf fa ff          call   403600 <runtime.chansend1>
138049          ch <- 20
138050    4576e6:       48 8b 44 24 20          mov    0x20(%rsp),%rax
138051    4576eb:       48 8d 1d ee 68 02 00    lea    0x268ee(%rip),%rbx        # 47dfe0 <runtime.egcbss+0xd>
138052    4576f2:       e8 09 bf fa ff          call   403600 <runtime.chansend1>
138053          add(ch)
138054    4576f7:       90                      nop
138055          a := <-ch
138056    4576f8:       48 c7 44 24 18 00 00    movq   $0x0,0x18(%rsp)
138057    4576ff:       00 00
138058    457701:       48 8b 44 24 20          mov    0x20(%rsp),%rax
138059    457706:       48 8d 5c 24 18          lea    0x18(%rsp),%rbx
138060    45770b:       e8 10 cc fa ff          call   404320 <runtime.chanrecv1>
138061          b := <-ch
138062    457710:       48 c7 44 24 18 00 00    movq   $0x0,0x18(%rsp)
138063    457717:       00 00
138064    457719:       48 8b 44 24 20          mov    0x20(%rsp),%rax
138065    45771e:       48 8d 5c 24 18          lea    0x18(%rsp),%rbx
138066    457723:       e8 f8 cb fa ff          call   404320 <runtime.chanrecv1>
138067  }
```

chan 在 runtime 中对应结构体 hchan, 其中比较直接的包括:
- elemtype 对应元素类型信息, 编译时产生
- elemsize 对应元素占用的内存大小
- dataqsiz 是最多可以缓存元素的数量, 即 `make(chan int, 2)` 中的 2
- closed 非 0 代表 chan 已被关闭

makechan 会直接申请需要的内存, 即变量 buf, 大小由单个元素的大小(elemsize)和缓存数量(dataqsize)决定.
元素会以环形队列的形式保存在 buf 中, qcount 是队列中元素的数量, sendx 是队首元素的坐标, recvx 是队尾元素的坐标.

将发送的元素保存在的缓存队列中代码 [chan.go#L216](https://github.com/golang/go/blob/go1.21.1/src/runtime/chan.go#L216):
```go
if c.qcount < c.dataqsiz {
    // Space is available in the channel buffer. Enqueue the element to send.
    qp := chanbuf(c, c.sendx)
    if raceenabled {
        racenotify(c, c.sendx, nil)
    }
    typedmemmove(c.elemtype, qp, ep)
    c.sendx++
    if c.sendx == c.dataqsiz {
        c.sendx = 0
    }
    c.qcount++
    unlock(&c.lock)
    return true
}
```
接收时直接从缓存队列中获取元素的代码 [chan.go#L537](https://github.com/golang/go/blob/go1.21.1/src/runtime/chan.go#L537):
```go
if c.qcount > 0 {
	// Receive directly from queue
	qp := chanbuf(c, c.recvx)
	if raceenabled {
		racenotify(c, c.recvx, nil)
	}
	if ep != nil {
		typedmemmove(c.elemtype, ep, qp)
	}
	typedmemclr(c.elemtype, qp)
	c.recvx++
	if c.recvx == c.dataqsiz {
		c.recvx = 0
	}
	c.qcount--
	unlock(&c.lock)
	return true, true
}
```

sendq 和 recvq 是两个 goroutine 队列, 前者保存了阻塞在发消息上的 goroutine, 后者保存了等待接收消息的 goroutine.
在发送和接收消息时, 都会优先判断是否由等待的 goroutine, 避免需要额外经过缓存中转一次.
```go
if sg := c.recvq.dequeue(); sg != nil {
    // Found a waiting receiver. We pass the value we want to send
    // directly to the receiver, bypassing the channel buffer (if any).
    send(c, sg, ep, func() { unlock(&c.lock) }, 3)
    return true
}
...
// Just found waiting sender with not closed.
if sg := c.sendq.dequeue(); sg != nil {
    // Found a waiting sender. If buffer is size 0, receive value
    // directly from sender. Otherwise, receive from head of queue
    // and add sender's value to the tail of the queue (both map to
    // the same buffer slot because the queue is full).
    recv(c, sg, ep, func() { unlock(&c.lock) }, 3)
    return true, true
}
```

如果发消息时即没有等待的接受者也没有可用的缓存空间,
则发送消息的 goroutine 会主动让处 CPU, 将自己的状态从 running 变为 waiting, 即对 gopark 的调用.
[chan.go#L237](https://github.com/golang/go/blob/go1.21.1/src/runtime/chan.go#L237):
```go
// Block on the channel. Some receiver will complete our operation for us.
gp := getg()
mysg := acquireSudog()
mysg.releasetime = 0
if t0 != 0 {
    mysg.releasetime = -1
}
// No stack splits between assigning elem and enqueuing mysg
// on gp.waiting where copystack can find it.
mysg.elem = ep
mysg.waitlink = nil
mysg.g = gp
mysg.isSelect = false
mysg.c = c
gp.waiting = mysg
gp.param = nil
c.sendq.enqueue(mysg)
// Signal to anyone trying to shrink our stack that we're about
// to park on a channel. The window between when this G's status
// changes and when we set gp.activeStackChans is not safe for
// stack shrinking.
gp.parkingOnChan.Store(true)
gopark(chanparkcommit, unsafe.Pointer(&c.lock), waitReasonChanSend, traceBlockChanSend, 2)
```
阻塞于发消息的 goroutine 需要等到其他接收消息的 goroutine 唤醒自己, 状态从 waiting 变化为 runnable 后, 才有机会继续执行,
即对 goready 的调用.
```go
func recv(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
	if c.dataqsiz == 0 {
		if raceenabled {
			racesync(c, sg)
		}
		if ep != nil {
			// copy data from sender
			recvDirect(c.elemtype, sg, ep)
		}
	} else {
		// Queue is full. Take the item at the
		// head of the queue. Make the sender enqueue
		// its item at the tail of the queue. Since the
		// queue is full, those are both the same slot.
		qp := chanbuf(c, c.recvx)
		if raceenabled {
			racenotify(c, c.recvx, nil)
			racenotify(c, c.recvx, sg)
		}
		// copy data from queue to receiver
		if ep != nil {
			typedmemmove(c.elemtype, ep, qp)
		}
		// copy data from sender to queue
		typedmemmove(c.elemtype, qp, sg.elem)
		c.recvx++
		if c.recvx == c.dataqsiz {
			c.recvx = 0
		}
		c.sendx = c.recvx // c.sendx = (c.sendx+1) % c.dataqsiz
	}
	sg.elem = nil
	gp := sg.g
	unlockf()
	gp.param = unsafe.Pointer(sg)
	sg.success = true
	if sg.releasetime != 0 {
		sg.releasetime = cputicks()
	}
	goready(gp, skip+1)
}
```

gopark 和 goready 我们后续介绍 GMP 的时候在展开.

共享内存和 chan 并没有优劣之分, 或者至少我认为二者没有谁是绝对更优.
chan 确实大幅简化了上层使用者的使用难度, 同时也依赖 goroutine 这样的轻量级协程.
