Istio 通过 k8s 的
[MutationWebhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
实现 为 Pod 动态注入 sidecar.
在集群中安装 istio controlplane 时, 会通过创建 mutationwebhookconfigurations, 使得 k8s 在创建 Pod 前先调用
controlplane 的 `/inject` 接口.

```json
k get mutatingwebhookconfigurations istio-sidecar-injector-1-21 -o json | jq '.webhooks[-1].clientConfig.service'
{
  "name": "istiod-1-21",
  "namespace": "istio-system",
  "path": "/inject",
  "port": 443
}
```

`/inject` 根据三方组合出 sidecar 的具体配置:
- 配置, 保存在 ConfigMap `istio-system/istio-sidecar-injector-1-21` 中
- 模板, 来自 [manifests](https://github.com/istio/istio/blob/1.21.0/manifests/charts/istio-control/istio-discovery/templates/istiod-injector-configmap.yaml).
- Pod 相关的信息

从注入的结果来看, sidecar 对应 [pilot/cmd/pilot-agent](https://github.com/istio/istio/blob/1.21.0/pilot/cmd/pilot-agent/main.go).
```json
k get -n test-ohai pods user-vz-6d7dd44c86-67jr4 -o json | jq '.spec.containers[0].args'
[
  "proxy",
  "sidecar",
  "--domain",
  "$(POD_NAMESPACE).svc.cluster.local",
  "--proxyLogLevel=warning",
  "--proxyComponentLogLevel=misc:error",
  "--log_output_level=all:info",
  "--log_as_json"
]
```

在大量初始化的工作之外, sidecar 的主要工作是启动 Envoy 并代理其到 controlplane 的请求.
```json
k exec -ti -n test-ohai user-vz-6d7dd44c86-67jr4 -c istio-proxy -- cat etc/istio/proxy/envoy-rev.json | jq '.static_resources.clusters[]
| select(.name == "xds-grpc") | .load_assignment.endpoints' -r
[
  {
    "lb_endpoints": [
      {
        "endpoint": {
          "address": {
            "pipe": {
              "path": "./etc/istio/proxy/XDS"
            }
          }
        }
      }
    ]
  }
]
```

这个 [XdsProxy](https://github.com/istio/istio/blob/1.21.0/pkg/istio-agent/xds_proxy.go#L626)
的核心目的在于处理链接 controlplane 时的证书问题.
暂时看起来, XdsProxy 在链接 Istiod 仅需要注入的 Server CA, 而不需要 Client 证书.
前者对应 ConfigMap `test-ohai/istio-ca-root-cert`.

Sidecar 另外一个经常需要用证书的场景是 [mTLS](https://istio.io/latest/docs/tasks/security/authentication/mtls-migration/).
这主要由 Envoy 的 Cluster 实现, 其通过
[Cluster.transport_socket_matches](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/cluster/v3/cluster.proto)
指明需要向 SDS(SecretDiscoveryService) 请求证书.
```json
istioctl pc cluster user-vz-69d9bff9b7-8zbnn.test-ohai --fqdn "outbound|80|vz|tasks.test-ohai.svc.cluster.local" -o json | jq '.[].transportSocketMatches[] | select(.name == "tlsMode-istio") | .transportSocket.typedConfig.commonTlsContext.tlsCertificateSdsSecretConfigs'
[
  {
    "name": "default",
    "sdsConfig": {
      "apiConfigSource": {
        "apiType": "GRPC",
        "transportApiVersion": "V3",
        "grpcServices": [
          {
            "envoyGrpc": {
              "clusterName": "sds-grpc"
            }
          }
        ],
        "setNodeOnFirstMessageOnly": true
      },
      "initialFetchTimeout": "0s",
      "resourceApiVersion": "V3"
    }
  }
]
```

这个 sds 依然是由 sidecar 代理的服务, 其入口在 [initSdsServer](https://github.com/istio/istio/blob/1.21.0/pkg/istio-agent/agent.go#L386).
一方面代理了 xDS 中 SDS 相关的请求, 另一方面请求 controlplane 获取相关证书.
```json
k exec -ti -n test-ohai user-vz-69d9bff9b7-8zbnn -c istio-proxy -- cat etc/istio/proxy/envoy-rev.json | jq '.static_resources.clusters[] | select(.name == "sds-grpc") | .load_assignment.endpoints'
[
  {
    "lb_endpoints": [
      {
        "endpoint": {
          "address": {
            "pipe": {
              "path": "./var/run/secrets/workload-spiffe-uds/socket"
            }
          }
        }
      }
    ]
  }
]
```
