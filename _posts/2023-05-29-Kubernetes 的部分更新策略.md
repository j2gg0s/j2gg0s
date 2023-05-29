事情的起因是使用 `kubectl apply` 时遇到的一个偶发的错误提示.
```bash
Warning: resource deployments/busybox-demo is missing the kubectl.kubernetes.io/last-applied-configuration annotation which is required by kubectl apply. kubectl apply should only be used on resources created declaratively by either kubectl create --save-config or kubectl apply. The missing annotation will be patched automatically.
```
为了快速规避可能的不良影响, 我转到了 `kubectl patch`.
在事后翻阅文档时, 可以明显的感受到 k8s 的 patch 基本考虑并处理了所有可能的场景, 极具借鉴意义.

部分更新, partial modification, 是一个常见而复杂的问题, HTTP PATCH 就是典型的部分更新语义,
一些实现会将 HTTP PUT 也实现成部分更新.
虽然业界对部分更新有充分而详细的讨论, 但很多实现者依然会忽略这些现成的结论而自行设计, 导致重复的问题重复出现.

## 部分更新的典型问题
我们假设已经存在一个资源 busybox-demo, 来讨论部分更新的典型问题.
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: busybox-demo
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: busybox
  template:
    metadata:
      labels:
        app: busybox
    spec:
      containers:
      - name: busybox
        image: busybox:1.28
        args:
        - sleep
        - "1000000"
```

部分更新的语义下, 我们只需要提供资源的标识和想要更新的字段.
所以如果我们希望把 replicas 更新为 2, 只需要提供如下信息.
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: busybox-demo
  namespace: default
spec:
  replicas: 2
```

而困难的点, 一方面在于如何把 replicas 更新为 0.
是仅需要把 replicas 设置为 0, 还是需要把 replicas 设置 null.
server 会不会错误的把 key 不存在理解为更新为 0?
很多实现没有明确而清晰的考虑零值的问题, 导致实现使用中出现歧义.

另一方面在于是否允许对数组进行部分更新? 怎么标识同一元素? 怎么标识某一元素的删除?
在 [JSON Merge PATCH](https://datatracker.ietf.org/doc/html/rfc7386) 中是无法对数组做部分更新,
这需要一些额外的工作.

### strategic
`kubectl patch` 提供了三种部分更新的策略, strategic 是默认选项.

你可以通过 `patch` 将 deploy 的 replicas 从 1 更新到 2.
```shell
➜ k get -n default pods
NAME                            READY   STATUS    RESTARTS   AGE
busybox-demo-7b8b5c46db-92x58   1/1     Running   0          3m15s
➜ k patch -n default deploy busybox-demo -p '{"spec": {"replicas": 2}}'
deployment.apps/busybox-demo patched
➜ k get -n default pods
NAME                            READY   STATUS    RESTARTS   AGE
busybox-demo-7b8b5c46db-92x58   1/1     Running   0          3m25s
busybox-demo-7b8b5c46db-j7grc   1/1     Running   0          4s
```

你可以通过指定 replicas 为 0, 将 pod 的数量减少到 0.
```shell
➜ k get -n default pods
NAME                            READY   STATUS    RESTARTS   AGE
busybox-demo-7b8b5c46db-96q2d   1/1     Running   0          60s
➜ k patch -n default deploy busybox-demo -p '{"spec": {"replicas": 0}}'
deployment.apps/busybox-demo patched
➜ k get -n default pods
NAME                            READY   STATUS        RESTARTS   AGE
busybox-demo-7b8b5c46db-96q2d   1/1     Terminating   0          79s
```

当你将 replicas 指定为 null 时, 有趣的事情发生了, replicas 变为了 1, 
这是 replicas 的默认值.
原因是 strategic 会将 null 视为要删除对应的 key, 进而导致其取默认值.
```shell
➜ k get -n default pods
NAME                            READY   STATUS    RESTARTS   AGE
busybox-demo-7b8b5c46db-kfhbw   1/1     Running   0          4s
busybox-demo-7b8b5c46db-r7znn   1/1     Running   0          4s
➜ k patch -n default deploy busybox-demo -p '{"spec": {"replicas": null}}'
deployment.apps/busybox-demo patched
➜ k get -n default pods
NAME                            READY   STATUS        RESTARTS   AGE
busybox-demo-7b8b5c46db-kfhbw   1/1     Running       0          21s
busybox-demo-7b8b5c46db-r7znn   1/1     Terminating   0          21s
➜ k get -n default deploy busybox-demo -o json | jq '.spec.replicas'
1
```

k8s 提供 strategic 的主要目的是为了实现数组的部分更新.
k8s 在文档中定义了两个注解: patchStrategy 和 patchMergeKey,
以 [PodSpec.Containers](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#podspec-v1-core) 为例.
patchStrategy 为 merge, 代表 k8s 会尝试将已有的数组和请求中的数组进行合并.
而 patchMergeKey 则指定了数组合并时用来判断数组元素是否相同的字段, 在此为 name.

所以, 我们可以通过如下的方式去修改和新增容器.
```shell
➜ k get -n default deploy busybox-demo -o json | jq '.spec.template.spec.containers[]'
{
  "args": [
    "sleep",
    "1000000"
  ],
  "image": "busybox:1.28",
  "imagePullPolicy": "IfNotPresent",
  "name": "busybox",
  "resources": {},
  "terminationMessagePath": "/dev/termination-log",
  "terminationMessagePolicy": "File"
}
➜ cat patch-containers.yaml
spec:
  template:
    spec:
      containers:
      - name: busybox
        resources:
          requests:
            cpu: 500m
      - name: busybox-add
        image: busybox:1.28
        args:
        - sleep
        - "3600"
➜ k patch -n default deploy busybox-demo --patch-file patch-containers.yaml
deployment.apps/busybox-demo patched
➜ k get -n default deploy busybox-demo -o json | jq '.spec.template.spec.containers[]'
{
  "args": [
    "sleep",
    "1000000"
  ],
  "image": "busybox:1.28",
  "imagePullPolicy": "IfNotPresent",
  "name": "busybox",
  "resources": {
    "requests": {
      "cpu": "500m"
    }
  },
  "terminationMessagePath": "/dev/termination-log",
  "terminationMessagePolicy": "File"
}
{
  "args": [
    "sleep",
    "3600"
  ],
  "image": "busybox:1.28",
  "imagePullPolicy": "IfNotPresent",
  "name": "busybox-add",
  "resources": {},
  "terminationMessagePath": "/dev/termination-log",
  "terminationMessagePolicy": "File"
}
```

我们也可以通过 patch 删除某个容器, 虽然好像 k8s 的文档里没有说这种做法.
```
➜ k get -n default deploy busybox-demo -o json | jq '.spec.template.spec.containers[].name'
"busybox"
"busybox-add"
➜ k patch -n default deploy busybox-demo -p '{"spec": {"template": {"spec": {"containers": [{"name": "busybox-add", "$patch": "delete"}]}}}}'
deployment.apps/busybox-demo patched
➜ k get -n default deploy busybox-demo -o json | jq '.spec.template.spec.containers[].name'
"busybox"
```

### JSON Merge Patch 和 JSON Patch
k8s 也支持 [JSON Merge Patch](https://datatracker.ietf.org/doc/html/rfc7386) 和 [JSON Patch](https://datatracker.ietf.org/doc/html/rfc6902).

相较于 strategic, JSON Merge Patch 的主要区别在于其并不支持数组层面的部分更新.
JSON Merge Patch 会直接使用请求中的数组替换现有的数组.
```shell
➜ k get -n default deploy busybox-demo -o json | jq '.spec.template.spec.containers[]'
{
  "args": [
    "sleep",
    "1000000"
  ],
  "image": "busybox:1.28",
  "imagePullPolicy": "IfNotPresent",
  "name": "busybox",
  "resources": {
    "requests": {
      "cpu": "500m"
    }
  },
  "terminationMessagePath": "/dev/termination-log",
  "terminationMessagePolicy": "File"
}
➜ cat patch-json.yaml
spec:
  template:
    spec:
      containers:
      - name: busybox-add
        image: busybox:1.28
        args:
        - sleep
        - "3600"
➜ k patch -n default deploy busybox-demo --patch-file patch-json.yaml --type merge
deployment.apps/busybox-demo patched
➜ k get -n default deploy busybox-demo -o json | jq '.spec.template.spec.containers[]'
{
  "args": [
    "sleep",
    "3600"
  ],
  "image": "busybox:1.28",
  "imagePullPolicy": "IfNotPresent",
  "name": "busybox-add",
  "resources": {},
  "terminationMessagePath": "/dev/termination-log",
  "terminationMessagePolicy": "File"
}
```

JSON Patch 则是完全的另外一个思路, 并不是完全的增量更新.
它允许对资源的任意字段单独进行复杂的操作, 更灵活强大的同时, 也更复杂.
```json
[
  { "op": "test", "path": "/a/b/c", "value": "foo" },
  { "op": "remove", "path": "/a/b/c" },
  { "op": "add", "path": "/a/b/c", "value": [ "foo", "bar" ] },
  { "op": "replace", "path": "/a/b/c", "value": 42 },
  { "op": "move", "from": "/a/b/c", "path": "/a/b/d" },
  { "op": "copy", "from": "/a/b/d", "path": "/a/b/e" }
]
```

## apply
当使用 kubectl 去操作资源时, 我们更推荐使用 apply 而不是 patch.
apply 做了一些额外工作, 极大的降低了使用成本.

当你使用 apply 去创建或者修改资源时, k8s 会通过特定的注解来记录这次请求.
```
➜ k get -n default deploy busybox-demo -o json | jq '.metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"]' -r | jq
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "annotations": {},
    "name": "busybox-demo",
    "namespace": "default"
  },
  "spec": {
    "replicas": 1,
    "selector": {
      "matchLabels": {
        "app": "busybox"
      }
    },
    "template": {
      "metadata": {
        "labels": {
          "app": "busybox"
        }
      },
      "spec": {
        "containers": [
          {
            "args": [
              "sleep",
              "1000000"
            ],
            "image": "busybox:1.28",
            "name": "busybox"
          }
        ]
      }
    }
  }
}
```

当你随后再通过 apply 去更新资源时, kubectl 会根据上次的请求, 当前的资源现状和这次的请求计算出具体应该如何修改.
而体而言:
- 如果某个字段在上次请求中存在, 但是在这次请求中不存在, 则被删除.
- 如果某个字段在当前请求中存在, 但是在现状中不存在或者值不同, 则被添加或者修改.

此时如果面临上文中需要删除某个容器的场景时, 直接在配置文件中删除对应的配置即可, 不再需要使用 $patch 这样的字段了.
需要注意, `patch` 等命令都不会更新这个注解, 所以如果最好不要把 apply 和其他命令混用在一个字段上.

## Reference
- [Update API Objects in Place Using kubectl patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#use-a-json-merge-patch-to-update-a-deployment)
- [How apply calculates differences and merges changes](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/#how-apply-calculates-differences-and-merges-changes)
- [Kubernetes Apply vs. Replace vs. Patch](https://blog.atomist.com/kubernetes-apply-replace-patch/)
