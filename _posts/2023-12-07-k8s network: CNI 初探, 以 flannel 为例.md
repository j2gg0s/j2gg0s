在过去的这些年, K8s 参与并推动了两个容器相关的标准.

一个是挂在 [OCI](https://opencontainers.org/) 下的 [runtime-spec](https://github.com/opencontainers/runtime-spec),
定义并规范了容器运行的方方面面. 依赖于此我们使用 K8s 是可以自由选择底层的容器运行时实现而不影响上层使用.

另一个是 [CNI (Container Network Interface)](https://github.com/containernetworking/cni),
定义了容器运行时如何管理容器网络.
在这个标准下开发的各类插件都可以直接在容器运行时中使用, 不再需要容器运行时进行适配.

## CNI 速览
CNI 中由两个关键角色, 一是容器运行时, 一是各类网络插件.
容器运行时负责在容器创建和销毁时, 根据配置调用指定插件的 ADD 和 DEL 命令.

同时, 比较有意义的点是:
- 插件以可执行文件的方式放置在宿主机的特定目录, 默认是 /opt/cni/bin
- 容器运行时通过环境变量向插件传递参数
- 插件通过 stdout 和 stderr 返回执行结果

为了避免重复实现, CNI 推荐插件通过调用其他插件来完成部分工作, 即 delegate.
同时, CNI 也为最基本的那部分需求提供了官方插件, 罗列于 [Plugins Overview](https://www.cni.dev/plugins/current/).

## flannel
Flannel 是一个简单但流行的容器网络方案, 其提供了一个 200 多行的 Yaml 文件来将自身部署到 K8s 中,
具体参见 [kube-flannel.yaml](https://github.com/flannel-io/flannel/blob/master/Documentation/kube-flannel.yml#L106).

部署的核心逻辑在于其中的 DaemonSet,
其一方面通过两个 initContaienrs 将 CNI Plugin 和配置文件安装到宿主机.
另一方面通过 flanneld, 处理路由转发逻辑.

### flannel-cni
容器运行时根据配置文件调用 [flannel-cni](https://github.com/flannel-io/cni-plugin) 后,
flannel-cni 会生成具体的配置文件调用其他插件来完成具体工作.

我们可以在宿主机上找到生成的具体配置信息:
```shell
~ cat /var/lib/cni/flannel/0e31e47db3bdb67088cd4b2369e641a8eea835208161177a02cbf6b8d26f3373 | jq
{
  "cniVersion": "0.3.1",
  "hairpinMode": true,
  "ipMasq": false,
  "ipam": {
    "ranges": [
      [
        {
          "subnet": "172.22.0.0/24"
        }
      ]
    ],
    "routes": [
      {
        "dst": "172.22.0.0/16"
      }
    ],
    "type": "host-local"
  },
  "isDefaultGateway": true,
  "isGateway": true,
  "mtu": 1500,
  "name": "cbr0",
  "type": "bridge"
}
```
这份配置文件依然遵循 CNI 的[配置规范](https://www.cni.dev/docs/spec/#section-1-network-configuration-format),
我们可以看出其调用了插件 [bridge](https://www.cni.dev/plugins/current/main/bridge/).
bridge 会将宿主机上的所有容器都链接到一个[虚拟交换机](https://wiki.archlinux.org/title/network_bridge)上.
同时也调用 [IPAM](https://www.cni.dev/plugins/current/ipam/) 在指定的网段内为容器分配了 IP.

### flannel daemon
[flannel](https://github.com/flannel-io/flannel) 以 DaemonSet 的方式运行在宿主机上.

我们以测试集群为例, 其配置保存 ConfigMap 中, 并且开启了 ipmasq.
```shell
~ k get -n kube-flannel cm kube-flannel-cfg -o json | jq '.data["net-conf.json"]' -r | jq
{
  "Network": "172.22.0.0/16",
  "Backend": {
    "Type": "host-gw"
  }
}
~ ps -ef | grep flanneld
root      49965  49079  0 8月01 ?       09:00:24 /opt/bin/flanneld --ip-masq --kube-subnet-mgr
root     224755 308100  0 12:11 pts/0    00:00:00 grep --color=auto flanneld
```

flanneld 的主要职责包括三个方面.
一是将提供给 flannel-cni 的配置文件保存到宿主机的文件系统, 其中包括了宿主机的 podCIDR.
```shell
~ cat /run/flannel/subnet.env
FLANNEL_NETWORK=172.22.0.0/16
FLANNEL_SUBNET=172.22.0.1/24
FLANNEL_MTU=1500
FLANNEL_IPMASQ=true
```

二是通过 iptables 为进出容器网络的流量做 SNAT.
```shell
~ iptables -t nat -L | grep "Chain FLANNEL-POSTRTG" -A 8
Chain FLANNEL-POSTRTG (1 references)
target     prot opt source               destination
RETURN     all  --  anywhere             anywhere             mark match 0x4000/0x4000 /* flanneld masq */
RETURN     all  --  172.22.0.0/24        172.22.0.0/16        /* flanneld masq */
RETURN     all  --  172.22.0.0/16        172.22.0.0/24        /* flanneld masq */
RETURN     all  -- !172.22.0.0/16        172.22.0.0/24        /* flanneld masq */
MASQUERADE  all  --  172.22.0.0/16       !base-address.mcast.net/4  /* flanneld masq */
MASQUERADE  all  -- !172.22.0.0/16        172.22.0.0/16        /* flanneld masq */
```
[MASQUERADE](https://askubuntu.com/questions/466445/what-is-masquerade-in-the-context-of-iptables) 的效果类似 SNAT,
其会修改数据包的来源地址.
Pod 访问外界网络的请求, 经过宿主机后, 来源地址会被修改宿主机, 因为 Pod IP 仅在容器网络内有效.

三是监听 node 信息后, 通过 ip route 将目标为宿主机上容器的数据包转发到宿主机.
172.22.1.0/24 是另一个节点的 podCIDR, 任何发送到相关容器的流量都会被转发给宿主机.
```shell
~ ip route | grep 172.22.1.0/24
172.22.1.0/24 via 10.30.180.55 dev eth0
~ k get nodes 10.30.180.55 -o json | jq '.spec.podCIDR' -r
172.22.1.0/24
```
