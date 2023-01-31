# gRPC over HTTP/2

## HTTP
HTTP 毫无疑问是互联网的基石之一.
HTTP/1.x 基于行的纯文本编码是其能够得到广泛应用的关键,
但同时带来的性能缺陷也早已被诟病已久.
HTTP/2 在完全兼容 HTTP/1.x 语法语义的基础上,
通过底层的改变, 带来性能的飞跃

[High Performance Browser Network](https://hpbn.co/http2/#brief-history-of-spdy-and-http2)
高质量的介绍了 HTTP/2 的实现和新特性.

![HTTP/2 binary framing layer](https://hpbn.co/assets/diagrams/ae09920e853bee0b21be83f8e770ba01.svg)
Binary framing layer 是 HTTP/2 引入的关键变更, 有三个核心概念:
- Stream
  抽象概念, 一个 TCP 链接上可以同时存在多个 stream.
- Message
  抽象概念, 对应完成一个完整的 HTTP 请求或响应. 由多个 frame 组成.
- Frame
  server/client 交换的最小单元, TODO

带来的一些核心收益和特性包括:

- 多路复用(Multiplexing)
HTTP/1.x 时代, 客户端在单个 TCP 链接中, 必须在收到上一个请求的响应后, 才能发送下一个请求.
这就导致, 如果客户端需要同时发送多个请求, 就需要创建多个 TCP 链接.
HTTP/2 借助 stream 的抽象, 可以在单个 TCP 链接中同时发送多个请求.

在这个基础上, 对于同一个目标, 客户端大多数时候也仅需要维护少量链接.

- Header Compression
通过在 server/client 之间维护已知 key 的霍夫曼编码,
并在传输过程中用霍夫曼编码而显著的减少 header 体积.

- 允许服务端主动推送响应

## gRPC over HTTP/2
gRPC 使用 HTTP/2, 而不是像 Thrift 使用 TCP, 做为传输层.
我们可以简单的认为,
protobuf 中的 rpc(unary) 会被映射成 HTTP 请求, frame 承载 protobuf 中的 message.
映射的规则可以参考
[gRPC over HTTP2](https://grpc.github.io/grpc/core/md_doc__p_r_o_t_o_c_o_l-_h_t_t_p2.html).
但如果你想清晰和正式的去理清这其中的关系时,
你会发现 grpc, protobuf, HTTP/2 这三者中存在大量同名不同意的核心概念.

值得注意的, streaming rpc 被映射到 HTTP/2 时,
HTTP/2 stream 的生命周期会拉长到 streaming rpc.
在使用类似 Envoy‘s HTTPConnectionManager 来对请求做额外工作时要避免阻塞.
