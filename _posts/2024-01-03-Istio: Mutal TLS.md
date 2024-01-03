基于 Istio 的 ServiceMesh 方案中支持 [Mutal TLS](https://istio.io/latest/blog/2023/secure-apps-with-istio/),
其实现依赖 Envoy 提供的 TLS 功能, [TLS Transport Socket](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/transport_sockets/tls/v3/tls.proto#tls-transport-socket-proto).

### Envoy
对于服务端(upstream)而言, 相关配置在 [listener.filter_chains[].transport_socket](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/listener/v3/listener_components.proto#envoy-v3-api-msg-config-listener-v3-filterchain),
对应 DownstreamTlsContext, 用于约束客户端需要满足的要求, 主要信息为公钥.

对于客户端(downstream)而言, 相关配置在 [cluster.transport_socket_match](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/cluster/v3/cluster.proto#envoy-v3-api-field-config-cluster-v3-cluster-transport-socket-matches),
对应 UpstreamTlsContext, 用于阐明链接服务端时的私钥和 SNI.

以实际应用为例, 首先通过 istioctl 获取某个 Pod 的相关配置并保存到 uw.json.
```shell
istioctl proxy-config all user-web-default-658548cdd-97sgl -o json > uw.json
```

由于集群运行在 PERMISSIVE 模式下, 即同时允许 PLAINTEXT 和 Mutal TLS, 所以 Istio 为服务的 gRPC 端口 50051 创建了两个监听.
```shell
~ cat uw.json | jq '.configs[] | select(.["@type"] | test(".*ListenersConfigDump")) | .dynamic_listeners[].active_state.listener | select(.name == "virtualInbound") | .filter_chains[-2:][].name'
"0.0.0.0_50051"
"0.0.0.0_50051"
~ cat uw.json | jq '.configs[] | select(.["@type"] | test(".*ListenersConfigDump")) | .dynamic_listeners[].active_state.listener | select(.name == "virtualInbound") | .filter_chains[-2:][].filter_chain_match'
{
  "destination_port": 50051,
  "transport_protocol": "tls",
  "application_protocols": [
    "istio",
    "istio-peer-exchange",
    "istio-http/1.0",
    "istio-http/1.1",
    "istio-h2"
  ]
}
{
  "destination_port": 50051,
  "transport_protocol": "raw_buffer"
}
~ cat uw.json | jq '.configs[] | select(.["@type"] | test(".*ListenersConfigDump")) | .dynamic_listeners[].active_state.listener | select(.name == "virtualInbound") | .filter_chains[-2:][] | keys'
[
  "filter_chain_match",
  "filters",
  "name",
  "transport_socket"
]
[
  "filter_chain_match",
  "filters",
  "name"
]
```
其中 TLS 相关的核心内容是申明使用动态获取的证书(ROOTCA)来校验客户端.
```shell
cat uw.json | jq '.configs[] | select(.["@type"] | test(".*ListenersConfigDump")) | .dynamic_listeners[].active_state.listener | select(.name == "virtualInbound") | .filter_chains[-2].transport_socket.typed_config.common_tls_context.combined_validation_context'
{
  "default_validation_context": {
    "match_subject_alt_names": [
      {
        "prefix": "spiffe://cluster.local/"
      }
    ]
  },
  "validation_context_sds_secret_config": {
    "name": "ROOTCA",
    "sds_config": {
      "api_config_source": {
        "api_type": "GRPC",
        "grpc_services": [
          {
            "envoy_grpc": {
              "cluster_name": "sds-grpc"
            }
          }
        ],
        "set_node_on_first_message_only": true,
        "transport_api_version": "V3"
      },
      "initial_fetch_timeout": "0s",
      "resource_api_version": "V3"
    }
  }
}
```

Cluster 的配置中同样也有两份, 分别针对使用和不使用 TLS.
```shell
~ cat uw.json | jq '.configs[] | select(.["@type"] | test(".*ClustersConfigDump")) | .dynamic_active_clusters[].cluster | select(.name == "outbound|50051|default|uc.dev.svc.cluster.local") | .transport_socket_matches[].name'
"tlsMode-istio"
"tlsMode-disabled"
~ cat uw.json | jq '.configs[] | select(.["@type"] | test(".*ClustersConfigDump")) | .dynamic_active_clusters[].cluster | select(.name == "outbound|50051|default|uc.dev.svc.cluster.local") | .transport_socket_matches[].match'
{
  "tlsMode": "istio"
}
{}
```
从文档上来看, Istio 为 endpoint 设置了相关的 metadata, 进而动态匹配 match, 使用对应的配置.
```shell
~ cat uw.json | jq '.configs[] | select(.["@type"] | test(".*EndpointsConfigDump")) | .dynamic_endpoint_configs[].endpoint_config | select(.cluster_name == "outbound|50051|default|uc.dev.svc.cluster.local") | .endpoints[].lb_endpoints[].metadata'
{
  "filter_metadata": {
    "istio": {
      "workload": "uc-default;dev;uc;;cn-shanghai"
    },
    "envoy.transport_socket_match": {
      "tlsMode": "istio"
    }
  }
}
```
具体看 downstream 的 TLS 配置, 其中申明了:
- 请求 SNI
- 请求加密的私钥证书为 default
- 使用公钥证书 ROOTCA 来校验服务端
- 限制了服务端证书中的 spiffe
```shell
~ cat uw.json | jq '.configs[] | select(.["@type"] | test(".*ClustersConfigDump")) | .dynamic_active_clusters[].cluster | select(.name == "outbound|50051|default|uc.dev.svc.cluster.local") | .transport_socket_matches[0].transport_socket.typed_config'
{
  "@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
  "common_tls_context": {
    "tls_params": {
      "tls_minimum_protocol_version": "TLSv1_2",
      "tls_maximum_protocol_version": "TLSv1_3"
    },
    "alpn_protocols": [
      "istio-peer-exchange",
      "istio",
      "h2"
    ],
    "tls_certificate_sds_secret_configs": [
      {
        "name": "default",
        "sds_config": {
          "api_config_source": {
            "api_type": "GRPC",
            "grpc_services": [
              {
                "envoy_grpc": {
                  "cluster_name": "sds-grpc"
                }
              }
            ],
            "set_node_on_first_message_only": true,
            "transport_api_version": "V3"
          },
          "initial_fetch_timeout": "0s",
          "resource_api_version": "V3"
        }
      }
    ],
    "combined_validation_context": {
      "default_validation_context": {
        "match_subject_alt_names": [
          {
            "exact": "spiffe://cluster.local/ns/dev/sa/default"
          }
        ]
      },
      "validation_context_sds_secret_config": {
        "name": "ROOTCA",
        "sds_config": {
          "api_config_source": {
            "api_type": "GRPC",
            "grpc_services": [
              {
                "envoy_grpc": {
                  "cluster_name": "sds-grpc"
                }
              }
            ],
            "set_node_on_first_message_only": true,
            "transport_api_version": "V3"
          },
          "initial_fetch_timeout": "0s",
          "resource_api_version": "V3"
        }
      }
    }
  },
  "sni": "outbound_.50051_.default_.uc.dev.svc.cluster.local"
}
```

### Istio
上述提及的两个证书都是通过 SDS 动态从 cluster sds-grpc 获取的, 其对应的地址为 /var/run/secrets/worload-spiffe-uds/socket.
```shell
~ cat uw.json | jq '.configs[] | select(.["@type"] | test(".*ClustersConfigDump")) | .static_clusters[].cluster | select(.name == "sds-grpc")'
{
  "@type": "type.googleapis.com/envoy.config.cluster.v3.Cluster",
  "name": "sds-grpc",
  "type": "STATIC",
  "connect_timeout": "1s",
  "load_assignment": {
    "cluster_name": "sds-grpc",
    "endpoints": [
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
  },
  "typed_extension_protocol_options": {
    "envoy.extensions.upstreams.http.v3.HttpProtocolOptions": {
      "@type": "type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions",
      "explicit_http_config": {
        "http2_protocol_options": {}
      }
    }
  }
}
```
Sidecar 监听这个端口并启动了一个 SDS 服务, 其会按需向 istiod 发起 Certificate Signing Request(CSR) 生成对应证书.

