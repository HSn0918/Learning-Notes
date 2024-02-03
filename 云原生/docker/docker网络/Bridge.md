#docker网络 
为主机 eth0 分配ip 192.168.0.101;
启动 docker daemon，查看主机 iptables ;
- POSTROUTING -A POSTROUTING -S 172.17.0.0/16 !-o docker0 -j MASQUERADE
在主机启动容器:
- docker run -d --name ssh -p 2333:22 centos-ssh
- Docker 会以标准模式配置网络;
	- 创建 veth pair ;
	- 将 veth pair的一端连接到 docker0 网桥;
	- veth pair 的另外一端设置为容器名空间的 eth0;
	- 为容器名空间的 etho分配ip;
	- 主机上的Iptables 规则 : PREROUTING -A DOCKER ! -i docker0 -p tcp -m tcp --dport 2333 -j DNAT --to-destination 172.17.0.2:22
![[Docker网络Bridge.jpg]]
>example
```
### Create network ns

```shell
mkdir -p /var/run/netns
find -L /var/run/netns -type l -delete
```

Start nginx docker with non network mode

```shell
docker run --network=none  -d nginx
```

Check corresponding pid

```shell
docker ps|grep nginx
docker inspect <containerid>|grep -i pid

"Pid": 876884,
"PidMode": "",
"PidsLimit": null,
```

Check network config for the container

```shell
nsenter -t 876884 -n ip a
```

 Link network namespace

```shell
export pid=876884
ln -s /proc/$pid/ns/net /var/run/netns/$pid
ip netns list
```

 Check docker bridge on the host

```shell
brctl show
ip a
4: docker0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
    link/ether 02:42:35:40:d3:8b brd ff:ff:ff:ff:ff:ff
    inet 172.17.0.1/16 brd 172.17.255.255 scope global docker0
       valid_lft forever preferred_lft forever
    inet6 fe80::42:35ff:fe40:d38b/64 scope link
       valid_lft forever preferred_lft forever
```

 Create veth pair

```shell
ip link add A type veth peer name B
```

 Config A

```shell
brctl addif docker0 A
ip link set A up
```

 Config B

```shell
SETIP=172.17.0.10
SETMASK=16
GATEWAY=172.17.0.1

ip link set B netns $pid
ip netns exec $pid ip link set dev B name eth0
ip netns exec $pid ip link set eth0 up
ip netns exec $pid ip addr add $SETIP/$SETMASK dev eth0
ip netns exec $pid ip route add default via $GATEWAY
```

 Check connectivity
```shell
 curl 172.17.0.10
```