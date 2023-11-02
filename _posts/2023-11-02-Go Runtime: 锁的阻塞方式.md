从极端简化的角度来看, 线程阻塞于锁的方式有两种:
- 自旋(spin)

  线程依然占有 CPU, 循环执行一些无意义的指令后, 检查状态, 如果可以持有锁则跳出循环.
  适合预期短时间内即可获得锁的等待场景, 虽然空耗了 CPU, 但是避免了线程的上下文切换.
- 睡眠(sleep)

  线程让出 CPU 的执行权, 进入等待队列. 可以持有锁后, 进入待执行队列.
  适合预期要阻塞一段时间的场景, 虽然有上下文切换, 但允许 CPU 执行其他任务, 提高了整体吞吐.

[runtime.lock2](https://github.com/golang/go/blob/go1.21.1/src/runtime/lock_futex.go#L80) 恰好涉及了上述两种场景.
```go
...
	// On uniprocessors, no point spinning.
	// On multiprocessors, spin for ACTIVE_SPIN attempts.
	spin := 0
	if ncpu > 1 {
		spin = active_spin
	}
	for {
		// Try for lock, spinning.
		for i := 0; i < spin; i++ {
			for l.key == mutex_unlocked {
				if atomic.Cas(key32(&l.key), mutex_unlocked, wait) {
					return
				}
			}
			procyield(active_spin_cnt)
		}

		// Try for lock, rescheduling.
		for i := 0; i < passive_spin; i++ {
			for l.key == mutex_unlocked {
				if atomic.Cas(key32(&l.key), mutex_unlocked, wait) {
					return
				}
			}
			osyield()
		}

		// Sleep.
		v = atomic.Xchg(key32(&l.key), mutex_sleeping)
		if v == mutex_unlocked {
			return
		}
		wait = mutex_sleeping
		futexsleep(key32(&l.key), mutex_sleeping, -1)
	}
...
```
在多核的情况下, lock2 在需要阻塞时会优先尝试进行 4 次自旋.
自旋的实现和 CPU 有关, amd64 下会使用到 [pause](https://www.felixcloutier.com/x86/pause.html) 这个指令.

[procyield](https://github.com/golang/go/blob/go1.21.1/src/runtime/asm_amd64.s#L775):
```
TEXT runtime·procyield(SB),NOSPLIT,$0-0
	MOVL	cycles+0(FP), AX
again:
	PAUSE
	SUBL	$1, AX
	JNZ	again
	RET
```

osyield 对应系统调用 [sched_yeild](https://man7.org/linux/man-pages/man2/sched_yield.2.html)
仅让出 CPU 使用权, 但线程依然在待执行队列等待调度.

futexsleep 对应睡眠, 底层系统调用是 [futex](https://man7.org/linux/man-pages/man2/futex.2.html).
调用成功后, 线程让出 CPU 使用权, 并变成等待状态,
直到 [unlock2](https://github.com/golang/go/blob/go1.21.1/src/runtime/lock_futex.go#L115)
中的 futexwake 被调用后, 线程才会重新放回待执行队列, 等到被调度后恢复执行.

Go 使用 GMP 做为调度模型, 所以在除了上述的自旋和睡眠外,
我们还可以选择仅挂起 G, 让线程(M) 去执行其他 G.
这样我们既可以避免线程级别的上下文切换成本, 又可以避免无意义的占用 CPU.

案例可以参考 [semacquire1](https://github.com/golang/go/blob/go1.21.1/src/runtime/sema.go).
goparkunlock 在释放 root.lock 之后, 将对应 G 的状态修改为 `_Gwaiting`, 并允许对应 M 执行其他 G.
```go
...
	for {
		lockWithRank(&root.lock, lockRankRoot)
		// Add ourselves to nwait to disable "easy case" in semrelease.
		root.nwait.Add(1)
		// Check cansemacquire to avoid missed wakeup.
		if cansemacquire(addr) {
			root.nwait.Add(-1)
			unlock(&root.lock)
			break
		}
		// Any semrelease after the cansemacquire knows we're waiting
		// (we set nwait above), so go to sleep.
		root.queue(addr, s, lifo)
		goparkunlock(&root.lock, reason, traceBlockSync, 4+skipframes)
		if s.ticket != 0 || cansemacquire(addr) {
			break
		}
	}
...
```
