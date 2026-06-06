#docker

相关笔记：[[kubernetes-basics]] | [[dockerfile]] | [[docker-commands]] | [[cgroup]] | [[namespace]] | [[union-fs]]

## Docker 概述

Docker 是基于 Linux 内核的 [[cgroup]]、[[namespace]] 以及 [[union-fs]] 等技术，对进程进行封装隔离的操作系统层面虚拟化技术。由于隔离的进程独立于宿主和其它隔离进程，因此也称其为容器。

- 最初基于 LXC 实现，从 0.7 版本开始去除 LXC，转而使用自行开发的 Libcontainer
- 从 1.11 版本开始进一步演进为使用 runC 和 Containerd
- 在容器的基础上进行了进一步封装（文件系统、网络互联、进程隔离等），使得 Docker 比虚拟机更轻便快捷

### Docker 架构

```mermaid
graph TB
    Client["Docker Client<br/>docker build / pull / run"]
    Daemon["Docker Daemon<br/>(dockerd)"]
    Registry["Docker Registry<br/>(Docker Hub / 私有仓库)"]
    
    Client -->|REST API| Daemon
    Daemon -->|pull / push| Registry
    
    subgraph Daemon内部
        Images["Images 镜像"]
        Containers["Containers 容器"]
        Networks["Networks 网络"]
        Volumes["Volumes 存储卷"]
    end
```

## 容器镜像

Docker 主要采取增量分发的策略来解决文件分发问题。

![[容器镜像.png]]

## Docker 的文件系统

典型的 Linux 文件系统组成：

- **Bootfs (boot file system)**
  - Bootloader — 引导加载 kernel
  - Kernel — 当 kernel 被加载到内存中后 umount bootfs
- **rootfs (root file system)**
  - `/dev`、`/proc`、`/bin`、`/etc` 等标准目录和文件
  - 对于不同的 Linux 发行版，bootfs 基本一致，但 rootfs 会有差别

### Docker 启动流程

```mermaid
graph TD
    A["加载 rootfs (readonly)"] --> B["完整性检查"]
    B --> C{"Linux vs Docker?"}
    C -->|Linux| D["切换为 readwrite"]
    C -->|Docker| E["union mount 挂载 readwrite 层"]
    E --> F["允许再次将下层 FS 设为 readonly 并向上叠加"]
    F --> G["形成多层 readonly + 1 层 writeable 的容器运行时态"]
```

- **Linux 启动**：将 rootfs 设置为 readonly，进行一系列检查，然后切换为 readwrite 供用户使用
- **Docker 启动**：将 rootfs 以 readonly 方式加载并检查，然后利用 union mount 将一个 readwrite 文件系统挂载在 readonly 的 rootfs 之上，每一个 FS 被称作一个 FS 层

### 写操作

由于镜像具有共享特性，对容器可写层的操作依赖存储驱动提供的写时复制和用时分配机制：

- **写时复制 (Copy-on-Write)**：一个镜像可以被多个容器使用，不需要在内存和磁盘上做多个拷贝。需要修改时，文件从镜像层复制到容器可写层进行修改，镜像里的文件不变。不同容器的修改相互独立、互不影响。
- **用时分配 (Allocate-on-Demand)**：按需分配空间，当文件被创建出来后才会分配空间。

[[docker-commands]]

### 容器存储驱动

![[容器存储驱动.png]]
![[容器存储驱动比较.png]]

#### 以 OverlayFS 为例

OverlayFS 是一种与 AUFS 类似的联合文件系统，属于文件级存储驱动，包含最初的 Overlay 和更新更稳定的 overlay2。

**Overlay 只有两层：upper 层和 lower 层。Lower 层代表镜像层，upper 层代表容器可写层。**

![[容器和overlayFS视图.png]]

```mermaid
graph TB
    subgraph "Container Mount (merged view)"
        merged["合并视图<br/>用户看到的文件系统"]
    end
    subgraph "Upper Layer (可写层)"
        upper["容器修改的文件"]
    end
    subgraph "Lower Layer (只读层)"
        lower["镜像文件"]
    end
    upper --> merged
    lower --> merged
```

查找逻辑：如果 upper 层有文件则直接使用，没有则从 lower 层拉取。

### OCI 容器标准

Open Container Initiative (OCI) 组织于 2015 年创建，致力于定义容器镜像标准和运行时标准。

OCI 定义了三个规范：
- **Runtime Specification (运行时标准)** — 定义如何解压应用包并运行
- **Image Specification (镜像标准)** — 定义应用如何打包
- **Distribution Specification (分发标准)** — 定义如何分发容器镜像

## Docker 网络

### 单主机网络

| 模式 | 说明 |
|------|------|
| [[network-null]] (`--net=none`) | 把容器放入独立的网络空间但不做任何网络配置，用户需手动配置 |
| Host (`--net=host`) | 使用主机网络命名空间，复用主机网络 |
| Container | 重用其他容器的网络 |
| [[network-bridge]] (`--net=bridge`) | 使用 Linux 网桥和 iptables 提供容器互联，通过 veth pair 连接 |

![[查看容器网络.jpg]]

### 跨主机网络

| 模式 | 说明 |
|------|------|
| Overlay (libnetwork, libkv) | 通过网络封包实现 |
| [[network-underlay]] | 使用现有底层网络，为每个容器配置可路由的网络 IP |
| Overlay | 通过网络封包实现 |

## 理解构建上下文 (Build Context)

- 运行 `docker build` 时，当前工作目录被称为构建上下文
- 默认查找当前目录的 Dockerfile 作为构建输入，也可通过 `-f` 指定：
  ```bash
  docker build -f ./Dockerfile
  ```
- 构建时首先会把构建上下文传输给 docker daemon，包含无用文件会导致传输时间长、资源消耗多、镜像体积大
- 可通过 `.dockerignore` 文件排除不需要的文件
- **最佳实践**：创建专门目录放置 Dockerfile，在该目录中运行 `docker build`

### Build Cache

构建镜像时，Docker 依次读取 Dockerfile 中的指令并按顺序执行。Docker 会先判断缓存中是否有可用的已存镜像，只有不存在时才会重新构建：

- 通常简单判断 Dockerfile 中的指令与镜像是否匹配
- 针对 `ADD` 和 `COPY` 指令，Docker 计算每个文件内容的 checksum 与现存镜像比较
- 其他指令（如 `RUN apt-get -y update`），简单比较指令字串是否一致
- **当某一层 cache 失效后，所有后续层级的 cache 均一并失效，后续指令都重新构建**

### 多段构建 (Multi-stage Build)

有效减少镜像层级的方式：

```dockerfile
# 第一阶段：编译
FROM golang:1.16-alpine AS build
RUN apk add --no-cache git
RUN go get github.com/golang/dep/cmd/dep

COPY Gopkg.lock Gopkg.toml /go/src/project/
WORKDIR /go/src/project/
RUN dep ensure -vendor-only

COPY . /go/src/project/
RUN go build -o /bin/project  # 只有这个二进制文件是产线需要的

# 第二阶段：运行
FROM scratch
COPY --from=build /bin/project /bin/project
ENTRYPOINT ["/bin/project"]
CMD ["--help"]
```

```mermaid
graph LR
    A["Stage 1: golang:1.16-alpine<br/>编译环境 ~800MB"] -->|"COPY --from=build"| B["Stage 2: scratch<br/>仅包含二进制 ~10MB"]
```

## [[dockerfile]]

常用指令速查：

| 指令 | 用途 | 示例 |
|------|------|------|
| `FROM` | 选择基础镜像（推荐 alpine） | `FROM [--platform=<platform>] <image>[@<digest>] [AS <name>]` |
| `LABEL` | 按标签组织项目 | `LABEL version="1.0" maintainer="dev@example.com"` |
| `RUN` | 执行命令 | `RUN apt-get update && apt-get install -y curl` |
| `CMD` | 容器默认运行命令 | `CMD ["executable","param1","param2"]` |
| `EXPOSE` | 声明端口 | `EXPOSE 80/tcp` |
| `ENV` | 设置环境变量 | `ENV KEY=value` |
| `ADD` | 复制文件（支持 URL 和解压） | `ADD [--chown=<user>:<group>] <src> <dest>` |
| `COPY` | 复制本地文件（推荐） | `COPY [--chown=<user>:<group>] <src> <dest>` |
| `ENTRYPOINT` | 容器入口命令 | `ENTRYPOINT ["executable","param1"]` |
| `VOLUME` | 定义外挂存储卷 | `VOLUME ["/data"]` |
| `USER` | 切换运行用户 | `USER <user>[:<group>]` |
| `WORKDIR` | 切换工作目录 | `WORKDIR /app` |

### ADD vs COPY 区别

- `COPY` 只支持本地文件复制，不支持 URL，不解压文件，语义更直白
- `ADD` 支持 URL 下载和自动解压本地压缩文件
- `COPY` 可用于多阶段构建：`COPY --from=build /bin/project /bin/project`
- **复制本地文件时优先使用 `COPY`**

### ENTRYPOINT 最佳实践

用 `ENTRYPOINT` 定义主命令，`CMD` 定义默认参数：

```dockerfile
ENTRYPOINT ["s3cmd"]
CMD ["--help"]
```

### 其他指令

`ARG` | `ONBUILD` | `STOPSIGNAL` | `HEALTHCHECK` | `SHELL`

## 面试要点

### 高频问题

**Q: Docker 容器和虚拟机（VM）的本质区别是什么？**
A: VM 通过 Hypervisor 虚拟化出完整硬件并运行独立的 Guest OS kernel，隔离强但启动慢、镜像 GB 级、占用资源多。Docker 是 OS 层虚拟化，所有容器共享宿主机 kernel，仅通过 namespace 做资源视图隔离、cgroup 做资源限额，因此秒级启动、镜像通常 MB 级。代价是隔离性弱于 VM，且容器无法运行与宿主机不同的 kernel（只能共享宿主 kernel）。

**Q: Docker 依赖哪些 Linux 内核技术实现隔离？**
A: 主要是 namespace（隔离视图，如 PID/Net/Mount/UTS/IPC/User namespace）、cgroup（限制 CPU/内存/IO 等资源）和 union FS（联合文件系统，实现分层镜像与写时复制）。namespace 解决"看不见"，cgroup 解决"用不超"，union FS 解决"镜像分层共享"。

**Q: Docker 镜像为什么要分层（layer）？写时复制是怎么工作的？**
A: 镜像由多个 readonly 层叠加，相同的基础层可被多个镜像/容器共享，节省存储并支持增量分发（只传缺失层）。容器启动时在只读层之上挂载一个 readwrite 层。当容器修改某个文件时，触发写时复制（Copy-on-Write）：先把文件从底层 readonly 层复制到容器可写层再修改，原镜像文件不变，不同容器互不影响；新建文件则走用时分配（Allocate-on-Demand），文件被创建出来时才分配空间。

**Q: OverlayFS（overlay2）的 lower、upper、merged 各代表什么？**
A: lower 层是只读的镜像层，upper 层是容器可写层，merged 是用户实际看到的合并视图。读取时先查 upper 层，命中就直接用，否则回到 lower 层拉取；写入和删除都在 upper 层完成（删除通过 whiteout 文件在 upper 层标记，使下层文件在 merged 视图中"消失"）。早期的 overlay 只支持单个 lower 层，overlay2 支持多个 lower 层、更稳定，是当前主流存储驱动。

**Q: ADD 和 COPY 有什么区别，生产中如何选择？**
A: COPY 只做本地文件/目录的复制，语义直白；ADD 额外支持远程 URL 下载和本地 tar 压缩包的自动解压。由于 ADD 行为隐式、容易踩坑，最佳实践是复制本地文件一律用 COPY，只有确实需要自动解压本地压缩包时才用 ADD。下载远程文件更推荐用 RUN curl/wget，便于在同一层校验与清理。

**Q: Docker 的 build cache 如何命中？为什么调整 Dockerfile 指令顺序能加速构建？**
A: 构建时逐条执行指令，每条先判断缓存中是否有匹配的已存镜像层。对 ADD/COPY 会计算文件内容 checksum 再比较，其余指令（如 RUN）只比较指令字符串是否一致。关键规则是：一旦某一层 cache 失效，其后所有层 cache 全部失效并重新构建。因此应把不常变动的内容（如依赖安装）放前面、频繁变动的源码 COPY 放后面，最大化缓存复用。

**Q: 多阶段构建（multi-stage build）解决了什么问题？**
A: 把"编译环境"和"运行环境"拆成多个 FROM 阶段，最终镜像只用 `COPY --from=<stage>` 拷贝产物（如 Go 二进制），编译器、源码、依赖统统不进最终镜像。这样既大幅缩小镜像体积（如 golang 编译镜像 ~800MB → 基于 scratch 的运行镜像 ~10MB），又减小攻击面，避免了过去用单层脚本拼命压缩镜像的繁琐做法。

**Q: 什么是 build context，为什么 .dockerignore 很重要？**
A: 执行 `docker build` 时，当前目录（context）会被整体打包发送给 docker daemon。如果目录里有 .git、node_modules、日志等无用文件，会拖慢传输、浪费资源，还可能因被 ADD/COPY 误带入而增大镜像。通过 .dockerignore 排除无关文件，并把 Dockerfile 放在专门的干净目录中构建，是推荐做法。

### 面试加分点

- 能讲清 Docker 的运行时演进：早期基于 LXC → 0.7 起改用自研 Libcontainer → 1.11 起拆分为 containerd + runC，并对应到 OCI 标准（runc 实现 Runtime Spec）。
- 熟悉 OCI 三大规范：Runtime Spec（怎么解压并运行容器）、Image Spec（怎么打包镜像）、Distribution Spec（怎么分发镜像），能说明这是镜像/运行时跨工具（Docker、containerd、Podman、CRI-O）互通的基础。
- 理解 Docker 架构是 C/S 模型：docker CLI 通过 REST API 与 dockerd 通信，daemon 管理 image/container/network/volume 并与 registry 交互，这解释了为什么可以远程操作 Docker。
- 能区分 bootfs 与 rootfs：bootfs（bootloader + kernel）在内核加载后被 umount，各发行版基本一致；镜像差异主要在 rootfs，这正是不同基础镜像（alpine/ubuntu）体积差异的来源。
- 掌握 ENTRYPOINT 与 CMD 的配合：ENTRYPOINT 定义固定主命令，CMD 提供可被命令行覆盖的默认参数；并能说明 exec 形式（JSON 数组）相比 shell 形式能让进程作为 PID 1 正确接收信号（如 SIGTERM）。
- 了解减小镜像体积的多种手段：选用 alpine/scratch 基础镜像、合并 RUN 指令减少层数、在同一层清理 apt/apk 缓存、配合多阶段构建，以及用 USER 切换非 root 用户提升安全性。
