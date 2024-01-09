最近在唤醒 Redis 相关的记忆, 以完成一次对公司内 Redis 相关资源使用情况的梳理和优化.

### 单线程
Redis 以单个线程处理所有客户端的请求.
这是基于非阻塞 IO 的多路复用, 结合 redis 的业务特点, 实现的超高效架构.

注意慢请求会严重伤害整体吞吐.

### Expire
从 [How Redis expires keys](https://redis.io/commands/expire/#how-redis-expires-keys) 来看,
Redis 现在有两种方式处理过期.

被动的方式, 是当获取 key 时再检查是否已过期.
仅依靠这种方式会导致过期的 key 因为一直未被访问而导致占用存储空间, 这在很早以前是一个问题,
知道 Redis 推出主动定期检查的方案.

这个主动的方案为了优化 ROI 显得有些奇怪, 且在某些场景下可能带来一些问题, 比如
[Improving key expiration in Redis @Twitter](https://blog.twitter.com/engineering/en_us/topics/infrastructure/2019/improving-key-expiration-in-redis).
默认的, 这个方案会每秒执行 10 次.
1. 随检选择 20 个 key, 检查并处理其中过期的 key.
2. 如果过期的 key 超过 25%, 则继续执行 1.

在 replication 和 AOF 中, 过期都对应具体的 DEL 命令.
master/replication 之间通过 DEL 通过过期时为了避免两个实例的时钟不同步,
但是 replication 依然会用本地时钟去避免客户端获取过期的 key.

### Evict
如果客户端写入新数据时, Redis 发现内存超过分配的值, 则会按配置的策略选择拒绝请求或者驱逐已有数据.
策略包括两个方面:
- 是否允许驱逐不带过期时间的 key, 允许的话是 allkeys, 不允许的话是 volatile
- 如何选择被驱逐的 key, 策略包括 lru, lfu, random, ttl 等
当然, 也可以选择 noeviction 来拒绝请求.

### 持久化: AOF 和 RDB
RDB 是对全部数据的快照, Redis 通过 fork 出一个进程, 将全部数据保存到文件系统.
其优势是数据紧凑, 同时不影响服务客户端请求. 但是需要注意:
- 在宕机时你会丢失部分数据
- fork 是阻塞的, 且可能因为数据量大而耗时颇久

AOF 类似 MySQL BinLog, 是通过追加的方式将命令保存到文件系统. 其特点是:
- 宕机时丢失数据的概率和数量, 远远小于 RDB
- Redis 现在支持自动重写(压缩) 日志, 减小占用体积
- 你可以通过 fsync 设置每次写入都同步到磁盘(always), 或者每秒同步一次(everysec), 或者不同步(no). fsync 的成本并不低.

### Replication
Redis 并不通过 AOF 来实现主从复制, 而是通过 slave 向 master 发送 PSYNC 命令来实现的.

理论上通过 (ReplicationID, offset) 这二元组记录主从复制的进度.
当 slave 重连是, master 如果发现对应 (ReplicationID, offset) 的命令还存在, 则直接触发增量同步,
否则要先进行全量同步.

[Redis replication](https://redis.io/docs/management/replication/) 中提到了一种可怕的场景:
- master 没有开启持久化
- master 重启, 内存数据清空
- master 同步到 slave, 导致 slave 的数据也清空

### 高可用或者水平扩容方案
1. Cluster
Redis 原生的集群方案, client 直接和每个分片链接.
client 向某个节点发送请求, 如果对应 key 在另外节点,
则该节点返回 MOVED, 由客户端重新向正确的节点发送请求.

也就是说节点并不负责代理和转发请求. 同时集群也无法处理跨多个分片的请求, 集群模式下不支持切换 DB(0-16).

向集群增加或者减少节点, resharding 由集群处理.

集群支持为每个分片配置 replication, 并支持 master 宕机后自动切换.

2. Sentinel
sentinel 做为一个哨兵监测 master/slave,
客户端首先向 sentinel 请求可用地址, 实现主从的自动切换.

3. [Client Ring](https://redis.uptrace.dev/guide/ring.html)
完全客户端的水平扩容方案.

4. [阿里云 Proxy](https://help.aliyun.com/zh/redis/product-overview/features-of-proxy-nodes)
支持跨分片的请求, 对客户端来说等价于单实例.
但请记住命运的所有馈赠早已标好了价格.

### [阿里云方案](https://help.aliyun.com/zh/redis/product-overview/product-introduction)
1. Tair 和 Redis, Tair 是阿里的 kv, 兼容 Redis 协议.
2. Redis 分为云原生和经典实例, 建议选择现在主推的前者.
3. Redis 支持主从, 集群和读写分离等功能, 价格依次增加
