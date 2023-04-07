k8s 的网络模型可以简单的划分为两部分:
- 允许通过 Service 访问背后的 Pod, 这部分功能有 kube-proxy 通过 ipvs/iptables 提供.
- 为 Pod 分配集群内唯一的 IP 并支持 Pod 相互之间的访问, 这部分功能有 [Network Plugins][] 提供.

Network Plugin 有很多的具体实现, Flannel 是其中一种, 以简单易用而出名.
其实现的主要功能包括三个方面:
- [flannel cni plugin][] 实现了 CNI 标准, 通过调用 [bridge plugin][] 和 [host-local plugin][], 根据 Node 的 podCIDR 为 Pod 分配唯一 IP.
- [flanneld][] 作为 agent, 监听 Node 变更事件, 配置 vxlan 等, 为 Pod 提供相互之间的访问能力.
- [flanneld][] 作为 agent, 为所在节点分配一段网络地址做为 podCIDR.

## 为 Pod 分配 IP
[CNI][] 是由 k8s 牵头指定的 specification,
规范了 k8s, container runtime 和 cni plugin 三者之间如何配置集群网络.
CNI 只定义了四个接口, ADD/DEL/CHECK/VERSION.
Pod 创建时, container runtime 会调用 cni plugin 的 ADD 接口为 Pod 配置网络, 包括虚拟网卡, IP 和路由等.
Pod 销毁时, container runtime 通过 cni plugin 的 DEL 接口回收之前分配的网络资源.

flannel 自身并没有实现这些功能, 只是实现 CNI 的接口, 将工作委托给 bridge 和 host-local.
bridge 和 host-local 都是 CNI 官方实现的 plugin, 避免第三方需要重复实现一些基本的功能.

### host-local plugin, IP Address Management [IPAM]
host-local 通过两个约束为 Pod 分配唯一的 IP.
首先, flanneld 会为每个节点分配一段不重叠的网段作为节点的 podCIDR.
当 Pod 被创建时, host-local 从节点的 podCIDR 中选择还未使用的 IP 分配给 Pod.
host-local 通过节点的文件系统来保存已分配的 IP 地址.

实际配置的例子和对应已分配地址的记录,
其中 `/var/lib/cni/flannel/{id}` 是 flannel cni plugin 保存的, 传递给 bridge 的配置,
`/var/lib/cni/networks/{name}` 是 host-local 用来存储已分配地址的文件夹.
```bash
# cat /var/lib/cni/flannel/577362af6ddde7acfade627730120025bdf6d5444913ae5a89a5ea3494f57bc6  | jq '.ipam'
{
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
}
# ls -alht /var/lib/cni/networks/cbr0/
总用量 32K
drwxr-xr-x. 2 root root 175 4月   4 20:50 .
-rw-r--r--. 1 root root  70 4月   4 20:50 172.22.0.30
-rw-r--r--. 1 root root  11 4月   4 20:50 last_reserved_ip.0
-rw-r--r--. 1 root root  70 4月   4 18:13 172.22.0.28
-rw-r--r--. 1 root root  70 4月   4 18:06 172.22.0.26
-rw-r--r--. 1 root root  70 4月   4 13:00 172.22.0.23
-rw-r--r--. 1 root root  70 4月   4 13:00 172.22.0.22
-rw-r--r--. 1 root root  70 3月  29 02:16 172.22.0.3
-rw-r--r--. 1 root root  70 3月  29 02:16 172.22.0.2
drwxr-xr-x. 3 root root  18 3月  29 02:16 ..
-rwxr-x---. 1 root root   0 3月  29 02:16 lock
```

### bridge plugin
bridge plugin 除了调用 IPAM 为 Pod 分配 IP 外,
主要是通过 [bridge][] 和 [veth][] 打通同一节点上 Pod 之间的网络.
每个 Pod 都拥有自己独立的 network namespace, 即使在同一节点上也无法直接访问.

bridge plugin 首先会在节点的网络空间创建一个 [bridge][],
随后为每个容器创建 [veth][], 链接 Pod 和节点的网络空间,
进而将节点上的所有 Pod 都链接到同一个 [bridge][], 实现互相之间的访问.
同时, bridge plugin 也会通过 host-local plugin 将 Pod 的 gateway 指定为 [bridge][] 的地址.

[bridge][] 的默认名称是 cni0.
```bash
# ifconfig cni0
cni0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1450
        inet 172.22.0.1  netmask 255.255.255.0  broadcast 172.22.0.255
        inet6 fe80::f883:1dff:fe22:bb5e  prefixlen 64  scopeid 0x20<link>
        ether fa:83:1d:22:bb:5e  txqueuelen 1000  [Ethernet]
        RX packets 19127952  bytes 6695305601 [6.2 GiB]
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 18943383  bytes 12675224666 [11.8 GiB]
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
```

Pod 的 gateway 被指定为 cni0.
```bash
# kubectl exec -n dev $POD -- ip route list -n
default via 172.22.0.1 dev eth0
```

Pod 和 cni0 通过 veth 关联.
```bash
# ip link show type veth
28: vethc782ee57@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP mode DEFAULT group default
    link/ether 62:b6:56:6e:f4:b8 brd ff:ff:ff:ff:ff:ff link-netnsid 0
29: vethb4e346a4@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP mode DEFAULT group default
    link/ether c2:6e:f2:fd:dd:9e brd ff:ff:ff:ff:ff:ff link-netnsid 1
49: veth3c3a686b@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP mode DEFAULT group default
    link/ether de:fc:55:e0:79:22 brd ff:ff:ff:ff:ff:ff link-netnsid 4
50: veth7286bf9b@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP mode DEFAULT group default
    link/ether 8e:3b:37:52:0c:84 brd ff:ff:ff:ff:ff:ff link-netnsid 5
53: vetheeb16977@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP mode DEFAULT group default
    link/ether fe:d2:42:fc:7a:eb brd ff:ff:ff:ff:ff:ff link-netnsid 2
55: vethc599277c@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP mode DEFAULT group default
    link/ether 42:91:8e:fa:c3:cf brd ff:ff:ff:ff:ff:ff link-netnsid 7
57: veth351a3fff@if2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue master cni0 state UP mode DEFAULT group default
    link/ether 26:7d:e4:35:12:8c brd ff:ff:ff:ff:ff:ff link-netnsid 3
```

对应的配置为:
```bash
# cat /var/lib/cni/flannel/577362af6ddde7acfade627730120025bdf6d5444913ae5a89a5ea3494f57bc6  | jq ''
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
  "mtu": 1450,
  "name": "cbr0",
  "type": "bridge"
}
```

### podCIDR
flanneld 会读取当前节点的 podCIDR 等信息并写到特定文件内, 供 flannel cni plugin 读取并传递给 bridge 和 host-local.
```bash
# cat /run/flannel/subnet.env
FLANNEL_NETWORK=172.22.0.0/16
FLANNEL_SUBNET=172.22.0.1/24
FLANNEL_MTU=1450
FLANNEL_IPMASQ=true
```

## 跨节点的 Pod 通信
同节点的 Pod 通信, 直接通过 [bridge][] 既可实现. 跨节点的 Pod 通信, 需要 flanneld 通过诸如 vxlan 之类的技术实现.

[host-local plugin]: https://www.cni.dev/plugins/current/ipam/host-local/
[bridge plugin]: https://www.cni.dev/plugins/current/main/bridge/
[flannel cni plugin]: https://www.cni.dev/plugins/v0.8/meta/flannel/
[flanned]: https://github.com/flannel-io/flannel
[bridge]: https://wiki.archlinux.org/title/network_bridge
[veth]: https://man7.org/linux/man-pages/man4/veth.4.html#:~:text=The%20veth%20devices%20are%20virtual,used%20as%20standalone%20network%20devices.
[Network Plugins]: https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/
[CNI]: https://www.cni.dev/
[vxlan]: https://support.huawei.com/enterprise/zh/doc/EDOC1100087027
