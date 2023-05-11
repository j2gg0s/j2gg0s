## volume 和 claim
在 k8s 中, 存储和 CPU/Memory 类似, 也是一种资源, 但其使用姿势却复杂的多.

用户申请存储资源的方式主要有两种: [Pod.spec.volumes](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#volume-v1-core)
和 [PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#persistentvolumeclaim-v1-core).
在申请后, 用户还需要通过 [Container.volumeMounts](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#container-v1-core) 指定如何挂载到容器.

对于用户的请求, k8s 首先会去管理员预先分配的 [PersistentVolume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#persistentvolume-v1-core) 中寻找符合的资源.
如果没有匹配的预分配资源, 则 k8s 会尝试通过 provision 机制创建并分配 [PersistentVolume][].

用户申请和资源匹配的核心在于 storageClass.
管理员为每一种支持的存储方案创建一个 [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/), 其中绑定了 provisioner 和相关参数.
用户在申请存储资源时, 需要在管理员提供的范围内, 申明预期的 storageClass.

## VolumePlugin: in-tree, flex 和 csi
kubelet 会监听 volume 相关的资源, 并通过 volume plugin 来实现存储资源的实际创建和挂载.

volume plugin 可以分为三大类: in-tree, flex 和 csi.

k8s 一开始直接在核心代码代码库中支持了一些常见的存储方案, 如 local, nfs 等.
这部分插件被称作 in-tree, 代码位于 [pkg/volume](https://github.com/kubernetes/kubernetes/tree/master/pkg/volume)

随着 k8s 的流行, 用户显然需要更多的存储方案.
如果依然是 in-tree 的形式来支持这些多样的第三方存储, 显然是不现实的.

k8s 起初提供的方案是 flex
Flex 要求管理员提前将 plugin 以可执行文件的形式放置在 Node 的指定目录下.
当需要处理对应存储的操作是 FlexVolume 去执行指定目录下的可执行文件.

一方面 flex 这套方案并不方便, 另一方面, k8s 一直是 Container XXX Interface 的主要推行者.
所以 k8s 在之后又接入了 Container Storage Interface(CSI), 并确定 CSI 会成为最终方案也并不奇怪.

第三方在实现了 CSI 后, 可以快速的接入 k8s. 以 [kubernetes-csi/csi-driver-nfs](https://github.com/kubernetes-csi/csi-driver-nfs) 为例:
- [external-provisioner](https://kubernetes-csi.github.io/docs/external-provisioner.html) 作为 sidecar, 提供了 provision.
- [node-driver-registrar](https://kubernetes-csi.github.io/docs/node-driver-registrar.html) 作为 sidecar, 自动完成了 [kubelet plugin]() 的注册.

## Reference
- [Kubernetes CSI Developer Documentation](https://kubernetes-csi.github.io/docs/)
- [Introducing Container Storage Interface (CSI) Alpha for Kubernetes](https://kubernetes.io/blog/2018/01/introducing-container-storage-interface/)
