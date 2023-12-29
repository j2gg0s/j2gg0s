最近在唤醒 Redis 相关的记忆, 我们准备在下个周期回顾, 调整并优化对 Redis 的使用.
预期的目标主要是将托管在阿里云的 Redis 迁移部分到云上/云下的 K8s 集群.

距离上次一本正经的了解 Redis 可能已经快过去 5 年, 这主要受益于:
- 我只是使用者而非维护者
- Redis 本身的质量很高, 稳定且能够轻易支持很高的吞吐, 使用者轻易不会遇到性能问题.
- 从饿了么时代开始, 我就相信保证系统稳定的关键是减少其不可降级的部分, 所以我们将 Redis 约束在缓存而非存储

### 基础
和 Nginx/Envoy 类似, Redis 也以非阻塞(non blocking)和多路复用(multiplexing)的形式使用 IO.
当请求的耗时主要在网络时, 非阻塞加上多路复用, 可以确保单个线程既可以处理大量的链接.
和 Nginx/Envoy 不同的点在于, Redis 只支持使用单个线程处理所有请求.
看上去会成为瓶颈, 实际上不是. 简单的说就是 Redis 内部操作很快.

Redis 清除过期 key 的方式有主动被动两种.
被动的方式是, 当 Redis 获取到 key 时, 如果发现其已经过期, 则删除.
主动是指, Redis 会定期的执行 [activeExpireCycle](https://github.com/redis/redis/blob/unstable/src/expire.c#L154), 默认是每秒 10 次.
主动清除中为了限制占用的资源, 会有较多的限制的, 简单理解就是: 取 20 个, 清除其中过期的, 如果过期的超过 25%, 则直接再来一次.

逐出 (Evict) 是指当有新的插入请求时, Redis 的内存已满, 导致需要拒绝插入请求或者删除已存在的数据.
逐出的策略是可以通过 maxmemory-policy 配置, 主要有两个纬度:
- allkeys/volatile: 是否仅对带过期时间的 key 执行逐出操作
- lru/lfu/ttl/random: 如何选择被逐出的 key

### Replication
Redis 的主从同步大体上和 MySQL 是类似的:
- 正常情况下, master 将增量的命令, 异步发送给 slave.
- 当 slave 重新链接到 master 时
    - 如果 master 判断可以恢复之前的增量同步到, 则 master 将未同步的命令发送给 slave.
    - 如果不可以, 则 master 会首先将当前的快照发送给 slave, 快照个格式毫无疑问是 RDB.

同步的进度由二元组 (ReplicationID, offset) 确定, ReplicationID 是 master 生成的随机字符串,
offset 是当前 ReplicationID 下已经同步数据的字节数.

slave 不会自行清除过期的 key, 而是 master 进行清除后会通过 DEL 指令同步给 slave.
但是当 key 过期时, 即使 slave 还未收到 master 同步过来的 DEL, 你也读取不到对应的内容.

### Cluster
