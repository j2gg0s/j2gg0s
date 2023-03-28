# [cpp] Learn cpp from LevelDB

[LevelDB]() 是一个用 cpp 实现的高性能 kv 存储库.
在阅读 [Envoy]() 之前, 我决定先阅读 [LevelDB]() 来学习并熟悉 cpp.

对于最近的写操作, LevelDB 在保存到内存中的 MemTable 前, 会先固化到磁盘日,
实现 write-ahead logging(WAL).

MemTable 的本质是 SkipList, 一种查询/插入的平均时间复杂度在 O(LogN) 的有序链表.
SkipList 通过典型的空间换时间的策略, 将数据在不同 level 重复存储实现 LogN 级别的查询.
具体的原理非常建议直接参考 [/wiki/Skip_list](https://en.wikipedia.org/wiki/Skip_list) 中的动图.

MemTable 的大小上限是 4MB, 超过则会被以 Table 的形式固化到磁盘.
Table 按升序存储 kv, 同时 Table 创建后不在允许变更.

LevelDB 以 Log-Structured Merge-Tree(LSM-Tree) 的形式来管理 Table.
LSM 下每个 Table 都属于某个 Level, 单个 Level 内的 Table 保存的 key range 不相交.
MemTable 固化的 Table 永远对应 Level0.  后台线程会进行 Compact,
将 LevelN 的 X 个文件和 LevelN+1 的 Y 个文件合并成 LevelN+1 的 Z 个文件.
当需要查询 key 对应的 value 时, 从 Level0 开始寻找, 如果找到则返回对应的 value.
如果没有找到, 则在下一个 Level 继续寻找.

LSM 的突出优势在于插入的时间复杂度为 O(1). LevelDB 特殊化了 Level0,
允许 Level0 的各个 Table 保存的 key range 存在相交的情况, 换取 MemTable 的固化仅批量写磁盘一次.
