K8s 的 apiserver 遵循 REST 风格, 在多年的发展和应用中解决了各种各样的实际问题,
是一个非常好的学习案例.

| Operation | HTTP Verb | URI(以 Deploy 为例)                                     | Status Code | 语义                   |
|-----------|-----------|---------------------------------------------------------|-------------|------------------------|
| CREATE    | POST      | /apis/apps/v1/namespaces/{namespace}/deployments        | 200/201/202 | 创建资源               |
| GET       | GET       | /apis/apps/v1/namespaces/{namespace}/deployments/{name} | 200         | 获取特定资源           |
| LIST      | GET       | /apis/apps/v1/namespaces/{namespace}/deployments        | 200         | 获取满足条件的所有资源 |
| REPLACE   | PUT       | /apis/apps/v1/namespaces/{namespace}/deployments/{name} | 200/201     | 替换特定资源           |
| PATCH     | PATCH     | /apis/apps/v1/namespaces/{namespace}/deployments/{name} | 200/201     | 修改特定资源的指定内容 |
| DELETE    | DELETE    | /apis/apps/v1/namespaces/{namespace}/deployments/{name} | 200/202     | 删除特定资源           |

如果有需要的话, 我们可以从状态码来区分写操作成功的实际含义.
- 对于 CREATE, 200 代表完全相同的资源之前就已经存在, 201 代表资源创建成功.
- 对于 REPLACE 和 PATCH, 201 代表资源之前不存在, 本次更新触发资源的创建.
- 对于 DELETE, 202 代表请求已被处理成功, 资源会在稍后被删除.
- CREATE 的 202 是在何种情况下返回的, 我并没有找到对应的代码.

六个操作中比较直观的是 CREATE/GET/DELETE, 没有太多展开的意义.

LIST 的复杂性来源于查询命中大量数据时的处理机制.

一种方式是基于偏移, 请求参数包括 offset, limit, 返回内容包括 offset, count.
在基于偏移的实现中, 终端用户可以知晓命中查询的资源总数, 能够直接跳转到第 N 页.
其缺点在于在大多数存储系统中, 比如 MySQL/PostgreSQL, 统计总数会带来额外成本, 跳转到第 N 页的插叙更可能是灾难性的.
起点的[完结榜](https://www.qidian.com/finish/)使用的就是基于偏移的分页方式.

另一种方式是基于游标, 请求参数包括 continue, limit, 返回内容包括 continue.
用户将上一次请求返回的 continue 做为下次请求的参数, 以实现翻到下一页的效果.
在基于游标的实现中, 返回内容大概率不包括面中查询的资源总数, 用户只能在当前的基础上翻到下一页.
抖音, 淘宝等的视频流都是下拉形式的游标分页.

K8s apiserver 选择的是游标分页.

替换(REPLACE)和更新(PATCH)并不是一回事, 前者知道用请求内容替换现有的整个资源, 后者指修改资源的一部分字段.
如果表达需要更新的字段也是一件复杂而令人纠结的事, K8s 提供了[三种方式](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/):
- `application/json-patch+json` 代表 [JSON PATCH](https://datatracker.ietf.org/doc/html/rfc6902)
JSON PATCH 具体指定了需要更新的字段, 优势在于无歧义, 但是在更新大量字段时显得过于繁琐.

假设数据如下:
```json
{
	"users" : [
		{ "name" : "Alice" , "email" : "alice@example.org" },
		{ "name" : "Bob" , "email" : "bob@example.org" }
	]
}
```
假设你想修改 Alice 的邮箱地址:
```json
[
	{
		"op" : "replace" ,
		"path" : "/users/0/email" ,
		"value" : "alice@wonderland.org"
	}
]
```

- `application/merge-patch+json` 代表 [JSON Merge Patch](https://datatracker.ietf.org/doc/html/rfc7386)
JSON Merge Patch 会将请求内容和已有资源合并(Merge), 合并的规则遵循 [JSON Merge Patch]().
其主要限制在于无法修改数组, 你只能覆盖整个数据而无法修改其中的一部分.

- `application/strategic-merge-patch+json` 是 K8s 特有的一种策略.
Strategic Merge Patch 依然是将请求内容和已有资源合并进而实现更新, 但其合并规则更为复杂的同时也更为强大.
K8s 在 [API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podspec-v1-core) 中为每个字段指定了合并策略, 默认为 replace.
也有一些数组字段被指定为 merge, 此时 K8s 还会指定 patch merge key, 即在数组中用于定位元素的 key.
比如 Pod 的 containers 就被指定为 merge, 且 key 是 name.
此时你可以通过如下内容修改特定 container 的 image 字段:
```yaml
spec:
  template:
    spec:
      containers:
      - name: patch-demo-ctr-2
        image: redis
```

在 Strategic 中, 你设置可以在请求中通过 $retainKeys 指定最终保留的 key 以实现删除效果.
假设资源如下:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: retainkeys-demo
spec:
  selector:
    matchLabels:
      app: nginx
  strategy:
    rollingUpdate:
      maxSurge: 30%
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: retainkeys-demo-ctr
        image: nginx
```
将发布策略修改为 Recreate 的请求内容如下:
```yaml
spec:
  strategy:
    type: Recreate
```
