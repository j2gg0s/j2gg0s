Dustin Specker 有个[Container Networking 系列的博客](https://dustinspecker.com/series/container-networking/),
手动模拟了 kube-proxy 在 iptables/ipvs 下的相关逻辑, 非常有助理解.
我们在这儿也从零走一遍 ipvs 相关的逻辑.

整个流程的大部分命令可以在 [Makefile](https://github.com/j2gg0s/j2gg0s/blob/main/examples/kube-proxy/Makefile) 中找到.

### 创建独立的网络命名空间(netns) foo

每个 Pod 都拥有自己独立的[网络命名空间](https://man7.org/linux/man-pages/man8/ip-netns.8.html#top_of_page), 进而实现网络隔离.
即使同一 Node 上的不同 Pod 之间也无法直接访问, Pod 也无法直接访问 Node 的网络.

```bash
ip netns add foo
ip netns exec foo ip link set dev lo up
```

### 在默认网络命名空间中创建 bridge, 并通过 veth, 将 foo 和默认空间链接起来

```bash
ip link add dev cni0 type bridge
ip addr add 172.22.0.1/24 dev cni0
ip link set dev cni0 up

ip link add dev veth_foo type veth peer name veth_foo_eth0
ip link set dev veth_foo master cni0
ip link set dev veth_foo up
ip link set dev veth_foo_eth0 netns foo
ip netns exec foo ip link set dev veth_foo_eth0 up
ip netns exec foo ip addr add 172.22.0.2/24 dev veth_foo_eth0
```

为了测试, 我们在 foo 中启动一个 http 服务.
```bash
~ ip netns exec foo python3 -m http.server -d foo 8080
Serving HTTP on 0.0.0.0 port 8080 (http://0.0.0.0:8080/) ...
```

这时候, 我们就可以直接访问命名空间 foo 中的 http 服务.
```bash
~ curl 172.22.0.2:8080 -s | grep im
<li><a href="im-foo">im-foo</a></li>
```

### 创建另一个网络命名空间 bar, 并通过 veth 关联到 bridge

```bash
ip netns add bar
ip netns exec bar ip link set dev lo up

ip link add dev veth_bar type veth peer name veth_bar_eth0
ip link set dev veth_bar master cni0
ip link set dev veth_bar up
ip link set dev veth_bar_eth0 netns bar
ip netns exec bar ip link set dev veth_bar_eth0 up
ip netns exec bar ip addr add 172.22.0.3/24 dev veth_bar_eth0
```

在测试 foo 和 bar 的互通之间, 我们需要先允许通过 cni0 来转发流量.
```bash
iptables -t filter -A FORWARD --in-interface cni0 -j ACCEPT
iptables -t filter -A FORWARD --out-interface cni0 -j ACCEPT

ip netns exec bar curl 172.22.0.2:8080 -s | grep im
<li><a href="im-foo">im-foo</a></li>
```

### 使用 ipvs 代理 foo 中的 http 服务

我们可以简单将 ipvs 理解为 L4 的负载均衡,
通过 ipvsadm 可以创建一个虚拟的 IP 地址, 并将相关流量转发给 foo.
```bash
ipvsadm --add-service --tcp-service 172.23.0.1:8080 --scheduler rr
ipvsadm --add-server --tcp-service 172.23.0.1:8080 --real-server 172.22.0.2:8080 --masquerading
```

在访问 172.23.0.1:8080 之前, 我们需要将 cni0 指定为 foo 的默认网关.
```bash
ip netns exec foo ip route add default via 172.22.0.1

~ curl 172.23.0.1:8080 -s | grep im
<li><a href="im-foo">im-foo</a></li>
```

### ipvs 的负载均衡功能

如果将 bar 也增加到 172.23.0.1:8080 的后端负载, 则可以免费体验 ipvs 的负载均衡功能.

```bash
ipvsadm --add-server --tcp-service 172.23.0.1:8080 --real-server 172.22.0.3:8080 --masquerading
ip netns exec bar ip route ad default via 172.22.0.1

➜ curl 172.23.0.1:8080 -s | grep im
<li><a href="im-foo">im-foo</a></li>
➜ curl 172.23.0.1:8080 -s | grep im
<li><a href="im-bar">im-bar</a></li>
➜ curl 172.23.0.1:8080 -s | grep im
<li><a href="im-foo">im-foo</a></li>
➜ curl 172.23.0.1:8080 -s | grep im
<li><a href="im-bar">im-bar</a></li>
```

### 在 Pod 中通过 Service 访问当前 Pod

k8s 中一个很经典的场景.
为了便于模拟, 我们先将 bar 中 172.23.0.1 的后端服务中踢出, 仅保留 foo.
```bash
ipvsadm --delete-server --tcp-service 172.23.0.1:8080 --real-server 172.22.0.3:8080
```

为了在 bar 中访问 172.23.0.1, 我们需要虚构对应的设备.
```bash
ip link add dev ipvs0 type dummy
ip addr add 172.23.0.1/32 dev ipvs0

ip netns exec bar curl 172.23.0.1:8080 -s | grep im
<li><a href="im-foo">im-foo</a></li>
```

但是你会发现, 我们无法直接在 foo 通过 ipvs 来访问 foo.
```bash
ip netns exec foo curl 172.23.0.1:8080 --connect-timeout 1
curl: (28) Connection timeout after 1001 ms
```

我们需要做三件事来解决这个问题:
- cni0 开启 promisc 来支持 hairpin
- 对来自 foo 的流量做一次 SNAT
- 激活 conntrack

```bash
ip link set cni0 promisc on
iptables -t nat -A POSTROUTING -s 172.22.0.0/24 -j MASQUERADE
sysctl net.ipv4.vs.conntrack=1 --write

~ ip netns exec foo curl 172.23.0.1:8080 -s | grep im
<li><a href="im-foo">im-foo</a></li>
```

MadHatter 在 severfault 上的[这个回答](https://serverfault.com/a/557776/1015156)非常好的解释了 hairpin 和这次 SNAT 的原因.

### TODO

有时间的话, 我们后续可以找个实际的集群, 来验证下上述的理解.
