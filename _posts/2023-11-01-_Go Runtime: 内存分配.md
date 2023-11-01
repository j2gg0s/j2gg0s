Go 的内存管理为了尽可能的高效, 做了很多精妙的设计.
这些精妙对诸如我这样的门外汉来说可能就是陌生和复杂了.

本文基于 go1.21.1 && linux && amd64 记录自己的理解.

Go 的内存管理来源于 [Thread-Caching Malloc(TCMalloc)](https://google.github.io/tcmalloc/design.html).

其核心思想是通过缓存降低内存分配的成本.

一方面, mheap 通过 mmap 向操作系统一次性申请大块内存, 32MB 起步.
另一方面, mheap 将这一大块内存以较小的单位, 一般而言是 8KB, 分配给 mcentral.

当需要为对象分配内存时,
首先根据对象大小确定对应哪种规格的 span.
如果本地的 P, GMP 中的 P, 缓存的 span 依然有空闲空闲, 则直接分配.
否则 P 需要首先从全局的 mcentral 处获取新的 span.

通过这套三级缓存, 大部分的内存申请操作都可以仅依赖本地的 P 完成,
少数情况下需要加锁后从 mcentral 处获取 span 填充到 P 的缓存,
极少数情况下需要从向操作系统申请内存.

源自 TCMalloc 的另一个主要优化是在一块连续的内存中用固定的大小保存对象, 对应 span.
Go 按元素占用的体积将 span 划分成了 60 多种规则, 比如说 8byte 及以下的都保存在 span0 中, 占用 8byte,
9~16byte 的都保存在 span1 中, 占用 16byte.
这种特性带来了时间和空间上的高效.
对, 虽然它可能因为对齐导致内存空间无法完全被占满, 但是考虑元信息的维护成本, 空间上依然是高效.
参见快速定位 span 空闲空间的例子 [nextFree](https://github.com/golang/go/blob/go1.21.1/src/runtime/malloc.go#L936):
```go
func (c *mcache) nextFree(spc spanClass) (v gclinkptr, s *mspan, shouldhelpgc bool) {
    ...
    v = gclinkptr(freeIndex*s.elemsize + s.base())
    ...
}
```

从 GC 相关的代码里面, 可以抠出上述对齐带来的一些优势.

首先, 从地址定位到所属 arena 的元信息, [arenaIndex](https://github.com/golang/go/blob/go1.21.1/src/runtime/mheap.go#L601)
```go
func arenaIndex(p uintptr) arenaIdx {
	return arenaIdx((p - arenaBaseOffset) / heapArenaBytes)
}
```
找到了 arena 后, 我们可以进一步找到地址所属的 span, [spanOfUncheck](https://github.com/golang/go/blob/go1.21.1/src/runtime/mheap.go#L717)
```go
func spanOfUnchecked(p uintptr) *mspan {
	ai := arenaIndex(p)
	return mheap_.arenas[ai.l1()][ai.l2()].spans[(p/pageSize)%pagesPerArena]
}
```
我们需要补充说明下, arena 会被分割成更小规模的 page, 8KB, 后再分配给 mcentral.
不同规格的 span 在程序启动前就确定会占用的 page 数量, 所以我们可以维护 page 到 span 的映射关系.

同时 arena 需要用 1bit 来标记每个地址是否是指针, GC 在标记阶段时可以据此快速确定是否可能存在引用关系.
