## kube-proxy 负责将目标为 Service 的流量转发给对应 Pod
kube-proxy 在 Linux 下有两种模式: iptables 和 ipvs, 其中 iptables 时默认选项.
相对 iptables, ipvs 在转发流量时有更低的延迟, 因为它使用哈希表, 而不是链表, 来存储 Service 和 Pod 的对应关系.
iptables 一样, ipvs 也依赖 [netfilter](https://en.wikipedia.org/wiki/Netfilter) 的钩子来处理数据包.

kube-proxy 通过监听 Service 和 EndpointSlice 的变更来更新 ipvs 规则.
测试集群中一个实际的例子:
```shell
~ k get -n dev svc signin
NAME     TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)                     AGE
signin   ClusterIP   172.23.83.19   <none>        50051/TCP,80/TCP,8080/TCP   13d
~ ipvsadm -ln -t 172.23.83.19:80
Prot LocalAddress:Port Scheduler Flags
  -> RemoteAddress:Port           Forward Weight ActiveConn InActConn
TCP  172.23.83.19:80 rr
  -> 172.22.0.253:8080            Masq    1      0          0
~ k get -n dev pods -l app=signin -o wide
NAME                            READY   STATUS    RESTARTS   AGE     IP             NODE           NOMINATED NODE   READINESS GATES
signin-test-a-8bcd5db69-9z2c5   2/2     Running   0          6d20h   172.22.0.253   10.30.180.56   <none>           <none>
```

## iptables 回顾
在探究 CNI 相关前, 我们先回顾下 iptables 的概念.

iptables 依赖 netfilter 来实现相关功能,
netfilter 是 Linux 提供的一系列和网络相关的钩子:
- NF_IP_PRE_ROUTING: 在路由前触发
- NF_IP_LOCAL_IN: 如果目标是本机, 则在路由后触发
- NF_IP_FORWARD: 如果目标不是本机, 则在路由后触发
- NF_IP_LOCAL_OUT: 从本机发布的流量触发
- NF_IP_POST_ROUTING: 本机发出或者转发的流量, 在 NF_IP_FORWARD/NF_IP_LOCAL_OUT 之后触发

iptables 的核心概念有 table, chain 和 rule.

5 个默认的 chain 对应 netfilter 的钩子:
- PREROUTING    -> NF_IP_PRE_ROUTING
- INPUT         -> NF_IP_LOCAL_IN
- FORWARD       -> NF_IP_FORWARD
- OUTPUT        -> NF_IP_LOCAL_OUT
- POSTROUTING   -> NF_IP_POST_ROUTING
用户也可以自定义 chain.

table 由 chain 组成, 按功能分为 5 个:
- filter: 过滤数据包
- nat: network address translation, 修改流量的来源或者目的地
- raw: 提供在 conntrack 之前处理数据包的机会
- mangle: 用来修改 IP Header, 我未关注
- security: 安全相关, 我未关注

conntrack 是一个维护链接状态的模块, 常见的状态包括:
- NEW 新建链接的包
- ESTABLISHED 包属于已建立的链接
- SNAT 来源地址被修改
- DNAT 目标地址被修改

chain 由 rule 组成, rule 分为 match 和 target 两个部分.
数据包遍历 chain 内的规则, 如果匹配上了规则的 match, 则执行 target.
target 分为两大类:
- 终止目标, 如 ACCEPT, RETURN, REJECT 等, 执行后控制权返回给 netfilter
- 非终止目标, 如 JUMP, MARK 等, 执行后继续执行 chain

MASQUERADE 是一个稍微复杂的 target, 它的作用类似 SNAT, 但无需指定特定 IP.
`iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE`
的含义就是将所有经过 eth0 出口的包的源 IP 替换.

## Container Network Interface
容器领域的两个重要标准: [Container Runtime Spec](https://github.com/opencontainers/runtime-spec) 和
[Container Network Interface](https://github.com/containernetworking/cni).
前者是容器运行的标准. 依赖于此, 诸如 K8s 这样的 runtime 上游可以允许使用者自主选择具体的实现, 比如 containerd, CRI-O 等.
后者约定了容器网络的标准. 依赖预期, 容器运行时允许使用者自主配置插件.

CNI 的核心功能是针对容器的创建分配 IP, 并支持容器之间的互相访问. 
其针对通用需求, 提供了一批基础的插件, [Plugins Overview](https://www.cni.dev/plugins/current/).
第三方也可以自行根据 CNI 实现插件, 提供更复杂的功能.

## Flannel 生效原理
[Flannel](https://github.com/flannel-io/flannel) 是一个相对简单的 CNI Plugin.
我们来简单的分析其生效原理.

测试集群的容器运行时是 containerd, 默认的:
- 配置文件位于 `/etc/containerd/config.toml`
- CNI Plugin 配置文件位于 `/etc/cni/net.d/`
- CNI PLugin 可执行文件位于 `/opt/cni/bin/`

测试集群的 CNI Plugin 配置文件如下:
```shell
cat /etc/cni/net.d/10-flannel.conflist
{
  "name": "cbr0",
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "flannel",
      "delegate": {
        "hairpinMode": true,
        "isDefaultGateway": true
      }
    },
    {
      "type": "portmap",
      "capabilities": {
        "portMappings": true
      }
    }
  ]
}
```
根据 [Network configuration format](https://www.cni.dev/docs/spec/#section-1-network-configuration-format),
我们可以知道上述配置文件指定了两个插件 flannel 和 portmap.

其中 [portmap](https://www.cni.dev/plugins/current/meta/portmap/) 是 CNI 自带的一个插件, 用于将宿主机的某个端口映射到容器端口.

hairpin 是网络领域的术语, 在 K8s 中对应场景如下: podA 访问 svcA 的流量被 kube-proxy 转发到 podA.

Flannel 的配置保存在 ConfigMap 中, 具体如下:
```shell
k get -n kube-flannel cm kube-flannel-cfg -o json | jq '.data["net-conf.json"]' -r
{
  "Network": "172.22.0.0/16",
  "Backend": {
    "Type": "host-gw"
  }
}
```
172.22.0.0/16 是集群的 Pod 网段.
[host-gw](https://github.com/flannel-io/flannel/blob/master/Documentation/backends.md#host-gw)
是 flannel 的一种模式, 其通过 iptables 将目标为容器的数据包直接转发到其所在的宿主机.
