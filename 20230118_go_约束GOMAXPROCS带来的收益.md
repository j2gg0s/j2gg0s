# [GoLang] 约束 GOMAXPROCS 带来的收益

GOMAXPROCS 限制了用来并发执行代码的系统线程数量, 默认等于 CPU 的数量.
使用 CPU 做为 GOMAXPROCS 的默认值, 是为了平衡执行效率和调度成本.
我们即希望有尽可能多的线程来提高并行程度, 又不希望系统线程被频繁的切换.
所以在假设基本独占机器资源的前提下, 我们选择 CPU 的数量做为默认的线程数量.

当我们的代码运行在 k8s 中, 基本面发生了一些改变.
一方面, 几十个应用共享一个宿主机(Node);
另一方面 Go 直接使用宿主机的 CPU 数量做为 GOMAXPROCS 的默认值.
于是出现了一些类似, 应用假设的 CPU 数量是 1, 但是默认创建了 64 个线程的情况.

这种 k8s 下的默认行为是否合理, 我们很难直接从代码中得出一个结论.
通过控制变量的方式来观察结果似一个不错的选择

![maxgoprocs_heavy_gc.png](./images/maxgoprocs_heavy_gc.png)
在 11:35 左右, 我们将应用的 GOMAXPROCS 主动设置为 4, 对比 14:30 和 11:30 的数据可以看到:

- go version: 1.17
- 线程数量从 49 下降到 13, 符合预期
- 受业务影响, QPS +13%
- 接口平均响应时间 -9%, CPU -5%
- GC 的耗时 -59%, STW(StopTheWorld) 的时间 -12%
- goroutine 在 Runnable 停留的时间 +50%

在一些负载较轻的应用中, 效果会更明显.
![maxgoprocs_gc.png](./images/maxgoprocs_gc.png)
