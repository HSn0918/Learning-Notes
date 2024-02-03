#docker
# Docker
- 基于Linux 内核的 [[Cgroups]]，[[Namespace]]，以及 [[Union FS ]]等技术，对进程进行封装隔离，属于操作系统层面的虚拟化技术，由于隔离的进程独立于宿主和其它的隔离的进程，因此也称其为容器。
- 最初实现是基于LXC，从0.7 以后开始去除 LXC，转而使用自行开发的 Libcontainer，从1.11开始则进一步演进为使用 runC和 Containerd。
- Docker 在容器的基础上，进行了进一步的封装，从文件系统、网络互联到进程隔离等等，极大的简化了容器的创建和维护，使得 Docker 技术比虚拟机技术更为轻便、快捷。
## 容器镜像
Docker主要采取增量分发的策略，解决文件分发的问题。
![[容器镜像.png]]
## Docker 的文件系统
典型的 Linux文件系统组成：
- Bootfs ( boot file system )
	- Bootloader-引导加载 kernel
	- Kernel-当 kernel被加载到内存中后umountbootfs
- rootfs ( root file system ) 
	- /dev，/proc，/bin，/etc 等标准目录和文件
	- 对于不同的 linux发行版bootfs基本是一致的但 rootfs 会有差别
### Docker 启动：
Linux
- 在启动后，首先将 rootfs 设置为 readonly,进行一系列检查,然后将其切换为“readwrite”供用户使用.
Docker启动
- 初始化时也是将 rootfs 以readonly 方式加载并检查，然而接下来利用 union mount 的方式将一个readwrite 文件系统挂载在 readonly 的rootfs 之上;
- 并且允许再次将下层的 FS (file system )设定为 readonly 并且向上叠加
- 这样一组 readonly 和一个 writeable 的结构构成一个container 的运行时态,每一个FS 被称作一个FS层。
### 写操作
由于镜像具有共享特性，所以对容器可写层的操作需要依赖存储驱动提供的写时复制和用时分配机制，以此来支持对容器可写层的修改，进而提高对存储和内存资源的利用率
- ==写时复制==
	写时复制，即 Copy-on-Write。一个镜像可以被多个容器使用，但是不需要在内存和磁盘上做多个拷贝在需要对镜像提供的文件进行修改时，该文件会从镜像的文件系统被复制到容器的可写层的文件系统进行修改，而镜像里面的文件不会改变。不同容器对文件的修改都相互独立、互不影响。
- ==用时分配==
按需分配空间，而非提前分配，即当一个文件被创建出来后，才会分配空间.
[[常用命令]]
### 容器存储驱动
![[容器存储驱动.png]]
![[容器存储驱动比较.png]]
#### 以OverlayFS为例
OverlayFS 也是一种与AUFS 类似的联合文件系统，同样属于文件级的存储驱动，包含了最初的 Overlay和更新更稳定的overlay2。
==Overlay 只有两层: upper 层和 lower 层，Lower 层代表镜像层，upper 层代表容器可写层.==
![[容器和overlayFS视图.png]]
假如上层目录没有，就拉取下层，反之。
### OCI容器标准
Open Container Initiative
- OCI组织于2015年创建，是一个致力于定义容器像标准和运行时标准的开放式组织
- OCI定义了镜像标准( Runtime Specification )、运行时标准(Image Specification )和分发标准( Distribution Specification )
	- 镜像标准定义应用如何打包
	- 运行时标准定义如何解压应用包并运行
	- 分发标准定义如何分发容器镜像
## Docker网络
### 单主机
>[[Null]](--net=None)
- 把容器放入独立的网络空间但不做任何网络配置;
- 用户需要通过运行 docker network 命令来完成网络配置
>Host
- 使用主机网络名空间，复用主机网络
>Container
- 重用其他容器的网络
>[[Bridge]](--net=bridge)
- 使用 Linux 网桥和 iptables 提供容器互联，Docker 在每台主机上创建一个名叫 docker0的网桥，通过 veth pair 来连接该主机的每一个EndPoint。
![[查看容器网络.jpg]]
---
### 跨主机
>Overlay(libnetwork, libkv)
- 通过网络封包实现。
>Remote(work with remote drivers)
- [[Underlay]]:
	- 使用现有底层网络，为每一个容器配置可路由的网络 IP。
- [[Overlay]]:
	- 通过网络封包实现
## 理解构建上下文( Build Context )
- 当运行docker build 命令时，当前工作录被称为构建上下文
- docker build 默认查找当前目录的 Dockerfile 作为构建输入，也可以通过-f指定 Dockerfile。
	- docker build -f ./Dockerfile
- 当docker build 运行时，首先会把构建上下文传输给 docker daemon，把没用的文件包含在构建上下文时会导致传输时间长，构建需要的资源多，构建出的镜像大等问题。
	- 试着到一个包含文件很多的目录运行下面的命令，会感受到差异;
	- docker build -f $GOPATH/src/github.com/cncamp/golang/httpserver/Dockerfile;
	- docker build $GOPATH/src/github.com/cncamp/golang/httpserver/;
	- 可以通过.dockerignore文件从编译上下文排除某些文件
- 因此需要确保构建上下文清晰，比如创建一个专门的目录放置 Dockerfile，并在目录中运行 docker build.
### Build Cache
构建容器镜像时，Docker 依次读取 Dockerfile 中的指令，并按顺序依次执行构建指令Docker 读取指令后，会先判断缓存中是否有可用的已存镜像，只有已存镜像不存在时才会重新构建.
- 通常 Docker 简单判断 Dockerfile 中的指令与镜像。
- 针对ADD和COPY指令，Docker判断该镜像层每一个文件的内容并生成一个checksum，与现存镜像比较时，Docker 比较的是二者的 checksum。
- 其他指令，比如 RUN apt-get -y update，Docker 简单比较与现存镜像中的指令字串是否一致。
- ==当某一层cache失效以后，所有所有层级的 cache 均一并失效，后续指令都重新构建镜像。==
### 多段构建( Multi-stage build )
- 有效减少镜像层级的方式
```dockerfile
FROM golang:1.16-alpine AS build
RUN apk add --no-cache git
RUN go get github.com/golang/dep/cmd/dep

COPY Gopkg.lock Gopkg.toml /go/src/project/
WORKDIR/go/src/project/
RUN dep ensure -vendor-only

COPY ./go/src/project/
RUN go build -o /bin/project ( 只有这个二进制文件是产线需要的，其他都是waste )

```
```dockerfile
FROM scratch
COPY --from=build /bin/project /bin/project
ENTRYPOINT["/bin/praject”]
CMD[--help]
```
## [[DockerFile]]
- FROM : 选择基础镜像，推荐 alpine
```dockerfile
FROM[--platform=<platform>] <image>[@<digest>][AS <name>]
```

- LABELS: 按标签组织项目
```dockerfile
LABEL multi.label1="value1"multi.label2="value2”other="value3"
配合label filter 可过滤镜像查询结果
docker images -f label=multi.label1="valuel"
```

- RUN : 执行命令
```dockerfile
最常见的用法是 
RUN apt-get update && apt-get install
这两条命令应该永远用&&连接，如果分开执行RUNapt-get update 构建层被缓存，可能会导致新 package 无法安装
```

- CMD : 容器镜像中包含应用的运行命令，需要带参数
```dockerfile
CMD ["executable","param1","param2"...]
```

- EXPOSE:发布端口
```dockerfile
EXPOSE <port>[<port>/<protocol>...]
```
	- 是镜像创建者和使用者的约定
	- 在 docker run -P 时，docker 会自动映射 expose 的端口到主机大端口，如0.0.0.0:32768->80/tcp

- ENV设置环境变量
```dockerfile
ENV <key>=<value> ...
```

- ADD:从源地址(文件，目录或者 URL)复制文件到目标路径
```dockerfile
ADD [--chown=<user>:<group>] <src>...<dest>
ADD [--chown=<user>:<group>]["<src>",..."<dest>"](路径中有空格时使用)
```
- ADD支持 Go 风格的通配符，如ADD check* /testdir/
- src 如果是文件，则必须包含在编译上下文中，ADD指令无法添加编译上下文之外的文件
- src 如果是URL
	- 如果 dest 结尾没有/，那么dest 是目标文件名，如果 dest 结尾有/，那么dest 是目标目录名
-  如果src是一个目录，则所有文件都会被复制至dest
- 如果src是一个本地压缩文件，则在ADD的同时完整解压操作
- 如果 dest 不存在，则ADD 指令会创建目标目录
- 应尽量减少通过ADD URL添加 remote 文件，建议使用 curl 或者 wget && untar
- COPY:从源地址(文件，目录或者URL)复制文件到目标路径

```dockerfile
COPY [--chown=<user>:<group>] <src>...<dest>
COPY [--chown=<user>:<group>]["<src>”..."<dest>"]// 路径中有空格时使用
```
	- COPY 的使用与ADD 类似，但有如下区别
	- COPY 只支持本地文件的复制，不支持URL
	- COPY不解压文件
	- COPY可以用于多阶段编译场景，可以用前一个临时镜像中拷贝文件
		- COPY --from=build /bin/project /bin/project
	- COPY 语义上更直白，复制本地文件时，优先使用COPY
	
- ENTRYPOINT:定义可以执行的容器镜像入口命令
```dockerfile
ENTRYPOINT ["executable”，"param1"，"param2"] // docker run参数追加模式
ENTRYPOINT command param1 param2 //docker run 参数替换模式
```
- docker run -entrypoint 可替换 Dockerfile 中定义的 ENTRYPOINT
- ENTRYPOINT的最佳实践是用ENTRYPOINT定义镜像主命令，并通过CMD定义主要参数如下所示
	- ENTRYPOINT["s3cmd"]
	- CMD[--help"]

- VOLUME:将指定目录定义为外挂存储卷，Dockerfile 中在该指令之后所有对同一目录的修改都无效
```dockerfile
VOLUME ["/data"] 
等价于docker run -v /data，可通过 docker inspect 查看主机的 mount point/var/lib/docker/volumes/<containerid>/_data
```

- USER:切换运行镜像的用户和用户组，因安全性要求，越来越多的场景要求容器应用要以non-root 身份运行
```dockerfile
USER <user>[:<group>]
```

- WORKDIR:等价于cd，切换工作目录
```dockerfile
WORKDIR /path/to/workdir
```

- 其他非常用指令
	- ARG
	- ONBUILD
	- STOPSIGNAL
	- HEALTHCHECK
	- SHELL