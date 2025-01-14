# **背景**

在XX项目中，客户没有容器相关基础知识，对于docker、containerd相关命令行工具不甚了解。相关开发工作都通过观云台进行操作，因此当客户训练完大模型打包成镜像，镜像大小为20G左右，基本只会通过观云台的镜像上传功能进行上传。

XX台原有版本上传镜像功能，受制于入口控制器及上层负载均衡的配置，上传大镜像都会因为超时失败，该问题严重阻塞了客户的正常开发流程，亟待解决。

# **难点**

1. 上传大文件如何保证不超时？
2. 如何保证上传文件的唯一性？
3. 上传过程如何保证断点续传？
4. XXk8s集群版本已升级到1.27，不再内置docker，如何进行适配？
5. 镜像仓库目前默认开启HTTPS，如何配置证书或者跳过证书验证？
6. 镜像上传过程涉及的目录是否支持挂载PVC进行统一存储管理？

# **方案**

大文件镜像上传，需要前后端一起协作才能解决。

## **接口拆分**

原有镜像上传功能中只提供一个用于流式上传的接口，同样也没有提供上传状态的信息。一旦失败，只能重新上传，而且也不清楚上传失败的具体原因，是在哪一步骤失败，使用体验非常不友好。

所以目前拆分成四个接口，用于优化使用体验。

1. 获取已上传切片的基本信息。
2. 上传文件切片。
3. 合并文件并推送镜像。
4. 获取合并文件、推送镜像状态。

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1729576312934-6057f9ef-a5c3-46b8-beba-93132b800082.png)

## **切片上传**

为了避免大文件上传，需要大文件进行最小单位进行切片上传，同时后端需要根据前端上传的参数按照规则对切片文件进行保存，目前方案采用的生成规则为 **文件名称_切片序号**。

### **切片方式**

切片有两种方式，该步骤需要前端完成切片。

1. 固定大小进行切片，并保存总切片个数，将每份切片与切片顺序相关联，当上传切片序号和总切片数相同时则表示上传完成。

相应参数

- 总切片数
- 切片序号
- 上传文件名称

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1729576312973-b28e94f3-77de-4ac3-8f82-d618339b3985.png)

1. 固定大小切片，保存每份切片的起始位置和偏移量，当起始位置+偏移量时等于总文件大小时则表示上传完成。

相应参数

- 总文件大小
- 切片的起始位置
- 切片的偏移量
- 上传文件名称

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1729576312969-fc50c276-d983-4186-996b-ed3f53dad1c7.png)

### **文件一致性**

切片上传虽然能够解决文件上传，但是无法保证文件一致性，因此需要信息摘要算法来保证，比如md5或者sha256，同时亦可以保存整个文件的md5值，在合并文件时做校验。具体流程如下。

![](https://cdn.nlark.com/yuque/0/2024/svg/46821905/1729576312838-ae33fe92-ae25-4429-a23c-1ba556cf353b.svg)

### **hash计算**

hash计算常见的有md5还有sha256，目前我们对该方案进行优化时主要采用md5的方式进行文件唯一性校验，但是计算md5比较耗时，需要在切片时就对切片进行md5计算。

可以使用对应的库进行优化，如使用 [spark-md5](https://www.npmjs.com/package/spark-md5)进行md5计算优化。

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1729576312957-b4b477ec-520d-47ba-a335-2acf00bab1d4.png)

### **控制上传过程**

前端上传切片的过程中，需要控制并发上传的数量，否则如果没有进行限制，容易浪费浏览器资源，导致卡死。

同时也需要提供暂停的选项，来保证关闭上传窗口时，停止上传的过程，因此可以使用先进先出带有容量的有界队列进行上传，具体实现逻辑如下。

![](https://cdn.nlark.com/yuque/0/2024/svg/46821905/1729576313267-b57efff8-bc47-4fb6-9fc2-1773a4838005.svg)

代码实现示例

```
function createPromiseQueue(size, maxRetries = 3) {
    const queue = [];
    const results = [];
    let activeCount = 0;
    let totalTasks = 0;
    let completedTasks = 0;
    let resolveAll;
    const allPromise = new Promise((resolve) => {
        resolveAll = resolve;
    });

    async function enqueue(promiseGenerator) {
        totalTasks++;
        while (activeCount >= size) {
            // 等待最早的 promise 完成
            await Promise.race(queue);
        }

        // 生成新的 promise 并加入队列
        activeCount++;
        let retries = 0;
        const attemptPromise = () => {
            return promiseGenerator()
                .then((result) => {
                    results.push(result);
                    return result;
                })
                .catch((error) => {
                    if (retries < maxRetries) {
                        retries++;
                        console.log("重新尝试", retries)
                        return attemptPromise(); // 重新尝试
                    } else {
                        console.error("超过最大重试次数", error)
                        throw error; // 超过最大重试次数，抛出错误
                    }
                })
                .finally(() => {
                    // 从队列中移除已完成的 promise
                    activeCount--;
                    completedTasks++;
                    queue.splice(queue.indexOf(promise), 1);
                    if (completedTasks === totalTasks) {
                        resolveAll(results);
                    }
                });
        };

        const promise = attemptPromise();
        queue.push(promise);
        return promise;
    }

    async function all() {
        await allPromise;
        return results;
    }

    return {
        enqueue,
        all,
    };
}

export default createPromiseQueue;
```

## **断点续传**

目前断点续传，主要是通过后端保存已经上传文件切片的序号和md5进行判断。

### **前端方案**

前端将每次上传成功的切片保存下来，如果最后完整的文件上传失败，则跳过已经上传的切片再上传之前失败的切片。

### **后端方案**

在前端每次切片上传的过程中，后端如果文件校验性通过则会将每份切片对应的hash值保存下来，每次前端进行上传时调用**获取已上传切片的基本信息**的接口，获取hash列表。上传时跳过已经在该列表中的切片，以此来达到断点续传的目的。

![](https://cdn.nlark.com/yuque/0/2024/svg/46821905/1729576313348-2598fb6f-c4c6-49f0-a871-f5ebc79263e6.svg)

## **合并文件**

合并文件包含将切片文件合并、加载文件为镜像、推送镜像这三个过程，这些操作是一个时间比较长的过程，所以该操作为异步操作，通过向 **Redis** 同步每个步骤的状态，再通过获取状态的接口从 **Redis** 中获取状态信息，直至最后的步骤完成。

![](https://cdn.nlark.com/yuque/0/2024/svg/46821905/1729576313408-84b5d350-d7b1-49e4-88cb-029c4dc7a63c.svg)

## **镜像工具选型**

上传的镜像文件，需要通过额外的工具将其加载为镜像，只有加载为镜像才能进行正常的推送，目前我们总共使用了三种工具进行尝试，分别是 docker、nerdctl、buildah，但是我们最终选择了buildah作为我们进行镜像操作的工具。

|   |   |   |   |
|---|---|---|---|
|**特性**|`Docker`|`nerdctl`|`Buildah`|
|**主要功能**|容器管理（镜像创建、容器运行等）|容器管理（与 Docker CLI 兼容）|镜像构建|
|**资源占用**|较高（需要 Docker Daemon）|较低（使用 `containerd`<br><br>）|较低（无 Daemon）|
|**CLI 兼容性**|官方 CLI 工具|与 Docker CLI 兼容|CLI 不兼容 Docker|
|**生态系统**|成熟的生态系统（Docker Compose、Swarm 等）|较少的生态系统|专注于镜像构建，不包含容器运行和管理|
|**安全性**|需要运行 Docker Daemon，可能有安全风险|无 Docker Daemon，安全性较高|无 Daemon 架构，安全性较高|
|**证书验证**|需要进行配置|需要进行配置|通过参数可跳过|
|**构建**|依赖本地docker-daemon<br><br>若在容器中运行需挂载socket文件|依赖containerd<br><br>多架构构建依赖buildkit后端<br><br>若在容器中运行需挂载socket文件|无需依赖，支持多架构|
|**是否支持k8s 1.27**|不支持|支持|支持|

## **为什么选择buildah**

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1729576313580-b68e175b-9cb2-4caf-b04a-334a0130c98b.png)

[containers/buildah: A tool that facilitates building OCI images. (github.com)](https://github.com/containers/buildah/tree/main)

1. buildah 不需要依赖后端进行构建，可以直接在容器中运行，方便集成，支持oci、docker的镜像格式。
2. 不依赖后端服务，所以资源占用较低。
3. 可以在容器中直接运行，所以支持在k8s 1.27 中以pod形式运行。
4. 构建和推送支持在命令行参数直接跳过证书验证，免去额外的配置。

### **构建buildah镜像**

alpine

```
FROM alpine:3.18
# 切换软件源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
# 安装必要依赖
RUN apk add iptables ip6tables cni-plugins containers-common fuse3 fuse-overlayfs buildah
# 配置使用overlayfs
RUN sed -i -e 's|#mount_program = "/usr/bin/fuse-overlayfs"|mount_program = "/usr/bin/fuse-overlayfs"|' /etc/containers/storage.conf
# 工作目录
WORKDIR /workspace
# 启动命令
ENTRYPOINT ["buildah"]
```

debian

```
FROM debian:11
# 配置软件源
RUN sed -i "s@http://\(deb\|security\).debian.org@http://mirrors.aliyun.com@g" /etc/apt/sources.list
# 安装对应依赖
RUN apt-get update && apt-get install -y fuse-overlayfs buildah
# 工作目录
WORKDIR /workspace
# 启动命令
ENTRYPOINT ["buildah"]
```

### **buildah使用**

[buildah/docs at main · containers/buildah (github.com)](https://github.com/containers/buildah/tree/main/docs)

1. 加载镜像

```
# 拉取镜像
buildah pull --tls-verify=false harbor.com/myregistry/test:v1
# 加载本地镜像
buildah pull docker-archive:./test-v1.tar
```

1. 构建镜像

```
# 构建镜像
# --tls-verify=false 跳过证书验证
buildah build --tls-verify=false -t harbor.com/myregistry/test:v1 -f Dockerfile .
```

1. 推送镜像

```
# 推送镜像到远端仓库
buildah push harbor.com/myregistry/test:v1
# 推送镜像到本地
buildah push  harbor.com/myregistry/test:v1 docker-archive:./test-v1.tar
```

1. 多架构构建

```
# 多架构构建
buildah build --jobs=2 --platform=linux/arm64/v8,linux/amd64 --manifest harbor.com/myregistry/test:v1 .
# 推送多架构信息到远端
$ buildah manifest push --all  harbor.com/myregistry/test:v1 docker://harbor.com/myregistry/test:v1
```

### **常见问题**

1. buildah 通过容器运行无法正常构建

**使用特权模式运行，要求网络管理、存储管理等权限。**

1. buildah 无法存储镜像

**配置下面的环境变量，使用兼容性更好的存储驱动。**

```
export BUILDAH_STORAGE=vfs
```

## **镜像存储挂载PVC**

目前buiildah经过测试排查，涉及到的工作目录有以下几个

- `/var/lib/containers` 镜像文件及相关配置。
- `/var/tmp`加载镜像产生的临时目录，加载完成则删除。

在底座环境中尝试将以上目录挂载到minio的PVC上，无法正常工作，排查发现挂载hostpath、emptyDir、nfs 都能正常工作，也查找了官方仓库的issue，可以确定minio的对象存储无法支持buildah存储镜像的文件格式，这个需要注意。

## **文件清理**

客户上传的镜像文件都是大文件，如果提供挂载的磁盘容量大小过小，比较容易写满，因此配置了对应的定时任务，每小时定时扫描上传文件目录中超过对应时限的文件，一旦超过时限则删除，目前设置的是6小时。

![](https://cdn.nlark.com/yuque/0/2024/svg/46821905/1729576313502-42358f53-0446-4231-905e-984fcac65542.svg)

# **遗留问题**

## **并发上传**

目前没有尝试过，多个用户同时进行上传大文件的场景，可能会遇到性能压力。

## **大文件上传的协议**

在博客中看到过，支持大文件上传的协议，有时间可以继续看下。