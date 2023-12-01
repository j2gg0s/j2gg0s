在大规模推进 Istio Multi Cluster 之前, 我先将 Istio 从 1.16 升级到了 1.20,
期间遇到了一些不在意料之中的小问题.

## 重复的 istio.stats
Istio 升级后, 手动检查 Envoy 配置的过程中, 我发现有重复的 istio.stats 插件存在.

istio.stats 是 Istio 添加到 Envoy 的一个 Filter Plugin,
用于生成 [Istio Standard Metrics](https://istio.io/latest/docs/reference/config/metrics/),
代码位于 [istio/proxy](https://github.com/istio/proxy/tree/master/source/extensions/filters/http/istio_stats).

随机选择一个幸运容器, 捕获 Listener 相关的配置:
```bash
istioctl proxy-config listener user-web-default-655ccdbf85-ftxx7 --port 8080 -o json > l.json
```
确实存在重复的 istio.stats 插件:
```bash
cat l.json | jq '.[].filterChains[].filters[].typedConfig.httpFilters[5:7][]'
{
  "name": "istio.stats",
  "typedConfig": {
    "@type": "type.googleapis.com/stats.PluginConfig",
    "metrics": [
      {
        "dimensions": {
          "method": "request.headers['content-type'].startsWith('application/grpc') ? request.url_path : response.headers['x-echo-pattern']"
        },
        "name": "request_duration_milliseconds"
      }
    ]
  }
}
{
  "name": "istio.stats",
  "typedConfig": {
    "@type": "type.googleapis.com/udpa.type.v1.TypedStruct",
    "typeUrl": "type.googleapis.com/stats.PluginConfig",
    "value": {}
  }
}
```

有些同学可能会疑惑, 为什么 @type 不同, 他们是对应同一个 Plugin 吗?
Envoy 处理非官方仓库内的扩展时, 通过 typedConfig["@type"] 来加载第三方注册的插件.
但当 type 是 Envoy 的通用结构 TypedStruct 时, 则使用 typedConfig.typeUrl 来加载插件,
参考 [Extension configuration (proto)](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/extension.proto).

Istio 允许通过 [Telemetry](https://istio.io/latest/docs/reference/config/telemetry/) 来定制 metrics.
从内容上可以看出第一个 istio.stats 来自我们自定义的 Telemetry:
```yaml
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: default
  namespace: istio-system
spec:
  metrics:
  - overrides:
    - match:
        metric: REQUEST_DURATION
      tagOverrides:
        method:
          operation: UPSERT
          value: 'request.headers[''content-type''].startsWith(''application/grpc'')
            ? request.url_path : response.headers[''x-echo-pattern'']'
    providers:
    - name: prometheus
```

而第二个 istio.stats 来自由 Istio 安装的 EnvoyFilter:
```bash
k get -n istio-system envoyfilters
NAME                           AGE
stats-filter-1.17-1-20-0       7m34s
stats-filter-1.18-1-20-0       22h
stats-filter-1.19-1-20-0       22h
stats-filter-1.20-1-20-0       22h
tcp-stats-filter-1.17-1-20-0   22h
tcp-stats-filter-1.18-1-20-0   22h
tcp-stats-filter-1.19-1-20-0   22h
tcp-stats-filter-1.20-1-20-0   22h
```
Istio 的默认配置 [profile/default.yaml](https://github.com/istio/istio/blob/release-1.20/manifests/profiles/default.yaml#L134)
中开启了 telemetry. 于是按 istioctl install 的时候会创建上面的 EnvoyFilters, 为对应的 Listener 添加 istio.stats 来捕获 metrics.

这二者并没有协作的很好, 添加了两个 istio.stats, 使得对应的 metrics 都会生成两个.
需要注意, 即使在后续升级过程中将 values.spec.telemetry.enabled 设置为 false, istio 也不会主动帮你删除这些 EnvoyFilters.

## fs.inotify 的限制
- 我们使用 helm v3 来管理发布. 默认情况下, helm 会将近期的历史纪律保存在 k8s secrets 中.
- istio 会监听 secret
- 我们的 istio sidecar 都是以用户 1337 运行的
- 我们测试环境的机器相对较大, 所以没台物理机可能同时存在几百个 pod

于是遇到了一个错误:
```
fatal Failed to start in-process SDStoo many open files
```
从 [issue/Failed to start in-process SDStoo many open files](https://github.com/istio/istio/issues/35829) 来看是需要增加允许同时监听的文件数量.

修改后也确实结局了这个问题.
