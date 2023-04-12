.PHONY: create-cni
create-cni:
	ip link add dev cni0 type bridge
	ip addr add 172.22.0.1/24 dev cni0
	ip link set dev cni0 up

.PHONY: create-foo
create-foo:
	ip netns add foo
	ip netns exec foo ip link set dev lo up

.PHONY: link-foo
link-foo:
	ip link add dev veth_foo type veth peer name veth_foo_eth0
	ip link set dev veth_foo master cni0
	ip link set dev veth_foo up
	ip link set dev veth_foo_eth0 netns foo
	ip netns exec foo ip link set dev veth_foo_eth0 up
	ip netns exec foo ip addr add 172.22.0.2/24 dev veth_foo_eth0

.PHONY: create-bar-and-link
create-bar-and-link:
	ip netns add bar
	ip netns exec bar ip link set dev lo up
	ip link add dev veth_bar type veth peer name veth_bar_eth0
	ip link set dev veth_bar master cni0
	ip link set dev veth_bar up
	ip link set dev veth_bar_eth0 netns bar
	ip netns exec bar ip link set dev veth_bar_eth0 up
	ip netns exec bar ip addr add 172.22.0.3/24 dev veth_bar_eth0

.PHONY: iptables-forward
iptables-forward:
	iptables -t filter -A FORWARD --in-interface cni0 -j ACCEPT
	iptables -t filter -A FORWARD --out-interface cni0 -j ACCEPT

.PHONY: ipvs-proxy-foo
ipvs-proxy-foo:
	ipvsadm --add-service --tcp-service 172.23.0.1:8080 --scheduler rr
	ipvsadm --add-server --tcp-service 172.23.0.1:8080 --real-server 172.22.0.2:8080 --masquerading

.PHONY: add-gateway
add-gateway:
	ip netns exec foo ip route add default via 172.22.0.1
	ip netns exec bar ip route add default via 172.22.0.1

.PHONY: ipvs-proxy-bar
ipvs-proxy-bar:
	ipvsadm --add-server --tcp-service 172.23.0.1:8080 --real-server 172.22.0.3:8080 --masquerading

.PHONY: create-ipvs
create-ipvs:
	ip link add dev ipvs0 type dummy
	ip addr add 172.23.0.1/32 dev ipvs0

.PHONY: support-hairpin
support-hairpin:
	ip link set cni0 promisc on

.PHONY: bridge-masq
bridge-masq:
	iptables -t nat -A POSTROUTING -s 172.22.0.0/24 -j MASQUERADE
