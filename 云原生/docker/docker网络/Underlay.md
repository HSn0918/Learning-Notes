#docker网络 
- 采用 Linux 网桥设备(sbrctl)，通过物理网络连通容器;
- 创建新的网桥设备mydr0;
- 将主机网卡加入网桥;
- 把主机网卡的地址配置到网桥，并把默认路由规则转移到网桥mydr0;
- 启动容器;
- 创建 veth 对，并且把一个 peer添加到网桥 mydr0;
- 配置容器把 veth 的另一个 peer 分配给容器网卡;
![[Docker网络Underlay.jpg]]