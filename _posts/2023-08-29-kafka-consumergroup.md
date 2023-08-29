Kafka 中的 topic 在物理上被划分成多个 partition, 每个 partition 独立存储在 broker 上, 并维护自己的 offset.
多个 consumer 组成 consumergroup 消费 topic 时, topic 的 partition 会被均分给 consumer.
Kafka 将均分的这部分逻辑放在了客户端, 理解可以参考历史文章
[Kafka Client-side Assignment Proposal](https://cwiki.apache.org/confluence/display/KAFKA/Kafka+Client-side+Assignment+Proposal)
和 [A Guide To The Kafka Protocol](https://cwiki.apache.org/confluence/display/KAFKA/A+Guide+To+The+Kafka+Protocol).
翻阅特定语言的客户端实现, 可以很好的帮助并验证自己的理解.

### FindCoordinator
在加入 consumergroup 之前, consumer 首先会根据 groupID 通过 FindCoordinatorRequest 确定某个 broker 做为 group 的 coordinator.
Coordinator 是针对 group 的, 使用单个 broker 做为 group 的 coordinator 可以大幅简化逻辑.

### JoinGroup
Consumer 会向 coordinator 发送 JoinGroupRequest.
Coordinator 收到请求后, 并不会立即返回. 而是等待一段时间, 以让 group 的所有 member 都能发出 JoinGroupRequest.

随后 coordinator 会随机选择一个 member 做为 leader, 让其负责分配 topic 下的 partition.

### SyncGroup
被选为 leader 的 consumer, 在收到的 JoinGroupResponse 中可以感知所有的 member.
基于此, 结合选择的分配策略, leader 会将分配结果通过 SyncGroupRequest 发送给 coordinator.

没有被选为 leader 的 consumer, 在收到 JoinGroup 的响应后也需要向 coordinator 发送 SyncGroupRequest.
Coordinator 会在收到 leader 的分配结果后, 再将其返回给 consumer.

### 频繁的 rebalance
大部分上了 k8s 的应用都会选择 hpa 来自动扩缩容, 但是扩缩容会导致 consumergroup 的成员增加或者减少,
带来 consumergroup 的 rebalance. **所以 consumergroup 应该尽量避免频繁的扩缩容.**

consumergroup 的成员数量大于 topic 的 partition 数量是没有任何意义的, 在某些实现下还会导致频繁的 rebalance.

### 非标使用
大多数情况下, 我们会以一个 consumergroup 去消费多个 topic.

当如果我们以多个 consumergroup 去消费相同的 topic 呢?
粗浅的理解下应该是可行, 因为 OffsetCommit 是以 consumergroup 为单位的.
```
v2 (supported in 0.9.0 or later)
OffsetCommitRequest => ConsumerGroup ConsumerGroupGenerationId ConsumerId RetentionTime [TopicName [Partition Offset Metadata]]
  ConsumerGroupId => string
  ConsumerGroupGenerationId => int32
  ConsumerId => string
  RetentionTime => int64
  TopicName => string
  Partition => int32
  Offset => int64
  Metadata => string
```
但实际是否可行, 或者是否会带来什么问题, 就不确定了.

或者, 我们为每个 consumergroup 创建一个 client, 然后以多个 consumergroup 去消费相同的 topic?
应该可以, 但是显然存在的问题是需要创建多个访问 kafka broker 的链接, 存在资源浪费问题.
