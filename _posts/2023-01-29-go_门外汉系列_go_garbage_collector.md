# [GoLang] 门外汉系列: Go Garbage Collector

### tiny/small/large
Go 为对象申请内存时会根据对象的大小分为三种情况: tiny(<=16byte)/small(<=32kb)/large(>32kb).
对于 tiny object, Go 允许多个 object 共用一个 slot.
对于 large object, Go 绕过 P's mcache 和全局的 mcentral, 直接从全局的 mheap 申请 span.
Small object 是最常见的情况, Go 设计了一套缓存机制来提高效率:

- P 拥有本地缓存的 span, Go 首先尝试从 P's mcache 中直接获取可用的内存
- 如果失败, 则尝试从全局的 mcentral 中获取 span.
- 如果 mcentral 是空的, 则会先从 mheap 中获取 span, 并填充到 mcentral.
- 如果无法从 mheap 中获取 span, 则 Go 会向系统申请一块较大的内存(arena).

### arena/span
Arena 是 Go 向系统申请内存的最小单位, 在 64 位非 windows 系统中为 64MB.
Span 是 Go Runtime 管理内存的单元, Small object 根据大小对应到 136 类 span.
同类 span 用固定的内存(elemsize)来存储单个对象.

### Mark-Sweep
Go 垃圾回收的核心思路是 Mark-Sweep:
第一阶段从根节点遍历并标记所有可访问的对象,
在第二阶段遍历所有内存并清除所有未被标记的对象实现空闲内存的回收.

Go 对每 PtrSize(8bytes) 的内存, 都使用 2bits 来标记: 是否需要扫描, 是否是指针.
在 mark 阶段, 按顺序扫描每 PtrSize 的内存, 如果是指针, 则将指向的 object 加入到待扫描队列中.
当不存在待扫描的内存时, 则可以进入 sweep 阶段.

同时 Go 对 span 中的每个对象都用 1bit 来标记是否在 mark 阶段被访问到.
在 sweep 阶段则根据这个标记来判断对应对象是否可以被回收.
