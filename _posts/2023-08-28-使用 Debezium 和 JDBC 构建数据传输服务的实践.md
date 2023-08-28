一开始, 我们直接使用阿里云的 [数据传输服务(Data Transimission Service, DTS)](https://www.aliyun.com/product/dts).
DTS 基于 Binlog 实时获取 MySQL 的数据变更, 并将其同步到的其他数据存储系统.

随着业务的扩张, 基于费用的考虑, 我们开始使用基于 Kafka Connect 的生态来实现 DTS 的功能,
并为业务提供 [Change Data Capture, CDC](https://en.wikipedia.org/wiki/Change_data_capture).

在大量的 connector 中, 我们选择:
- 通过 [debezium](https://debezium.io/documentation/reference/2.3/index.html) 将 MySQL 的数据变更实时导入 kafka
- 通过 [kafka-connect-jdbc](https://docs.confluent.io/kafka-connectors/jdbc/current/sink-connector/overview.html) 从 kafka 消费 [debezium]() 生产的消息并写入 MySQL, 阿里云ADB 等系统.

本文后续介绍一些常见的坑.

## MySQL
在选型之初出于谨慎和保守选择了 MySQL 5.7.

### Timestamp, Datetime and Timezone
无论是 Timestamp 还是 Datetime, MySQL 实际存储时都不会携带任何时区信息, 这是我们首先要明确的.
我们可以在和 MySQL 的链接中指定时区, 默认是 MySQL 服务端的时区.

对于 Timestamp, MySQL 存储的是 UTC 对应的值. 在更新时, MySQL 会将时间从链接对应的时区转换到 UTC 后存储。
在读取是, MySQL 会将存储的值从 UTC 转换到链接对应的时区后返回.

对于 Datetime, MySQL 不会在读取或更新时, 根据链接时区进行任何转换.

### 字符集
为了能够正确的存储表情等特殊字符, 在 MySQL 中我们应该使用 utf8mb4 以上的字符集.
由于历史原因, MySQL 中的 utf8, 实际是 utf8mb3, 最多只包括 3bytes, 并没有办法完整的存储 Unicode Characters.

## Debezium

### Timestamp
Debezium 会将 Timestamp 转换成 STRING.
在后续的消费中, JDBC 需要额外的代码才能识别出 Timestamp.

### Datetime
Debezium 通过 `time.precision.mode` 控制将 Datetime 转换成什么类型.
- `adaptive_time_microseconds` 可以保留较高的精度, 但是因为是 Debezium 自定义的类型, 所以无法直接被 JDBC 识别.
- `connect` 仅能保留 3 位小数, 但是使用 Kafka Connect 原生类型, 可以直接被 JDBC 识别.

### Schema Registry
在没有 Schema Registry 的情况下, 我们可以选择 JSON Converter, 并将 Schema 直接保存在每个消息中.

也可以选择使用 [apicurio] 做为 kafka connect 的 schema registry.
此时我们可以选择将消息格式保存为 avro, 并且不用在消息中冗余 schema.
和冗余 schema 的消息体积相差在 30 倍以上.

## kafka-connect-jdbc
mysql-connector-java:5.1.49 中
- characterEncoding=UTF-8 指定链接使用字符集为 utf8mb4, 并且 5.1.47 以前的版本没有办法指定 utf8mb4.
- 确定是否要禁止 useLegacyDatetimeCode

debezium 会将 Timestmap 的默认值 `CURRENT_TIMESTAMP` 映射为 `1970-01-01T00:00:00Z`, 需要注意:
- `1970-01-01T00:00:00` 和 `1970-01-01 00:00:00` 在 MySQL 中是等价的, 都是合法的.
- Timestamp 理论上的最小值是 `1970-01-01 00:00:01`, 并不包括 `1970-01-01 00:00:00`, 需要额外的代码处理下.

## kafka
CDC/DTS 这种场景中很适合将 `cleanup.policy` 设置为 compact.
数据库的每行记录仅会对应一条消息, 并不会因为反复更新而膨胀.

同时通过 `compression.type` 开启消息的压缩, 消息消耗的体积略小于数据库实际大小, 我们可以考虑对消息进行永久保存.
方便某些需要频繁拉取全量数据的消费者.
