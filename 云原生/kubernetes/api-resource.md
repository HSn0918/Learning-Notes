# **1.背景**

目前版本迭代需要对不同集群下的api-resource资源进行分类

以下是 Kubernetes API Resources 数据结构的层次与分类的格式化说明：

Kubernetes API Resources 数据结构层次与分类

1. 根级别：API Resources 表示整个集群支持的 API 资源的集合。
2. 分组（Group） 每个资源按照功能域归类为一个 Group。  
    • 示例：  
    • core（核心资源组）  
    • apps（应用资源组）  
    • batch（批处理任务）  
    • networking.k8s.io（网络相关资源）  
    • custom（自定义资源组）
3. 版本（Version） 每个 Group 可以包含多个 API 版本，表示资源的不同稳定性阶段。  
    • 示例：  
    • v1（稳定版本）  
    • v1beta1（测试版本）
4. 资源种类（Kind） 每个 Version 包含具体的资源类型（Kind）。  
    • 示例：  
    • Pod  
    • Deployment  
    • Job
5. 资源范围（Scope） 每个资源根据使用范围分为以下两类：  
    • Cluster：集群级别的资源（不受命名空间限制）。  
    • 示例：Node、Namespace  
    • Namespaced：命名空间级别的资源。  
    • 示例：Pod、Service
6. 特殊资源类型 一些资源属于特殊的 Workload 类型，用于运行用户应用程序。  
    • 示例：  
    • Deployment  
    • StatefulSet  
    • DaemonSet

**数据结构的具体描述**

以下是一个具体的示例数据结构：

```
{
  "apiResources": [
    {
      "group": "core",
      "versions": [
        {
          "version": "v1",
          "resources": [
            { "kind": "Pod", "scope": "Namespaced" },
            { "kind": "Node", "scope": "Cluster" },
            { "kind": "Service", "scope": "Namespaced" }
          ]
        }
      ]
    },
    {
      "group": "apps",
      "versions": [
        {
          "version": "v1",
          "resources": [
            { "kind": "Deployment", "scope": "Namespaced", "type": "Workload" },
            { "kind": "StatefulSet", "scope": "Namespaced", "type": "Workload" }
          ]
        },
        {
          "version": "v1beta1",
          "resources": [
            { "kind": "ReplicaSet", "scope": "Namespaced", "type": "Workload" }
          ]
        }
      ]
    }
  ]
}
```

**数据结构字段说明**

|   |   |   |
|---|---|---|
|**字段**|**类型**|**说明**|
|apiResources|数组|包含所有 API Resources 信息|
|group|字符串|资源所属的功能域|
|versions|数组|该 Group 下支持的 API 版本|
|version|字符串|API 版本号，如 v1、v1beta1|
|resources|数组|该版本下的资源类型列表|
|kind|字符串|资源的种类，例如 Pod、Node|
|scope|字符串|资源范围：Cluster 或 Namespaced|
|type|字符串（可选）|字符串（可选）|

树状结构图表：

```
Kubernetes API Resources
├── Group: core
│   └── Version: v1
│       ├── Pod (Namespaced)
│       ├── Node (Cluster)
│       └── Service (Namespaced)
├── Group: apps
│   ├── Version: v1
│   │   ├── Deployment (Workload, Namespaced)
│   │   └── StatefulSet (Workload, Namespaced)
│   └── Version: v1beta1
│       └── ReplicaSet (Workload, Namespaced)
```

# **2.用例描述**

- 用户点击资源中心-Kubernetes资源的时候，支持查看GVK列表
- 用户点击资源中心-Kubernetes资源-GVK，支持查看资源实例
- 用户点击资源中心-Kubernetes资源-GVK-资源实例，支持编辑资源实例

# **3.详细设计**

![](https://cdn.nlark.com/yuque/0/2024/svg/46821905/1732086152684-761a0487-aef9-4240-85ac-c751a31d3c10.svg)

## **3.1 api-resource资源**

在 Kubernetes（K8S）中，GVK（Group - Version - Kind）是一种用于唯一标识 Kubernetes API 对象的三元组。它在整个集群的资源管理和操作中起到关键作用，用于准确地定位和操作各种资源类型，通过它可以准确地对集群中的各种资源进行定义、操作和管理。

在进入K8S资源页面的时候，通过当前集群获取所有api-resource，利用gv进行api-resource的分类,存入map中。利用api-resource里面的NameSpace参数进行判断是集群的还是命名空间的资源。

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1732086153204-eaf23780-3d31-4e7d-92eb-39d28c422702.png)![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1732086153115-dc92b941-49ef-4b47-a850-e7be66e03bec.png)

- 通过该资源信息，可以结合字段属性使用 Kubernetes 客户端获取具体的资源实例：

1. 根据 `namespace.isNamespace` 判断资源是否属于命名空间范围。若值为 `true`，则通过 `namespace.name` 提取命名空间名称，限定资源的查询范围。
2. 根据 `workload.isWorkload` 判断资源是否为工作负载类型。若值为 `true`，可通过 `workload.ready` 获取当前就绪的副本数，结合 `workload.desired` 表示的期望副本数，分析资源的实际运行状态是否达到预期。

```
{
  "name": "apiserver", // 资源名称，标识该资源的唯一名称
  "namespace": { 
    "isNamespace": true, // 是否为命名空间范围的资源，true 表示属于某命名空间，false 表示为集群级别资源
    "name": "caas-system" // 命名空间名称，仅在 isNamespace 为 true 时有效
  },
  "workload": {
    "isWorkload": true, // 是否为工作负载类型资源，true 表示为 Deployment、StatefulSet 等需要副本管理的资源
    "ready": 1, // 当前已就绪的副本数，表示资源实例的运行状况
    "desired": 1 // 期望的副本数，用于定义资源的目标状态
  },
  "createTime": "2006-01-02 03:04:05" // 资源创建时间，标准时间格式，用于记录资源的创建时间
}
```

命名空间资源工作负载

```
{
  "name": "apiserver",
  "namespace": {
    "isNamespace": true,
    "name": "caas-system"
  },
  "workload": {
    "isWorkload": true,
    "ready": 1,
    "desired": 1
  },
  "createTime": "2006-01-02 03:04:05"
}
```

命名空间资源

```
{
    "name": "configmap-loader",
    "namespace": {
      "isNamespace": true,
      "name": "kube-system"
    },
    "workload": {
      "isWorkload": false,
      "ready": 0,
      "desired": 0
    },
    "createTime": "2006-01-04 08:15:42"
  },
```

集群资源

```
{
  "name": "cluster-role-binding",
  "namespace": {
    "isNamespace": false,
    "name": ""
  },
  "workload": {
    "isWorkload": false,
    "ready": 0,
    "desired": 0
  },
  "createTime": "2006-01-07 14:23:56"
}
```

见原型图：

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1732086152991-9624ffec-0eca-4f3a-ac6d-a751fc4b9b68.png)

根据命名空间，集群，负载均衡展示

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1732086153118-a8694d8a-9b35-43d4-b915-b6de5efc6ee3.png)

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1732086153171-befd818f-a9df-4936-ba1c-c51f07081de6.png)

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1732086153769-c5ccb296-a692-4b91-8c58-7543c5265689.png)

## **3.2 yaml**

方案：在创建和编辑 YAML 文件时，前端将 YAML 格式的数据转换为 JSON，并将其传递到后端。后端通过 Kubernetes 提供的序列化工具，将接收到的 JSON 数据解析并序列化为 Kubernetes 对象，以完成资源的更新或创建操作。前端传输的 JSON 示例如下：

```
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "example-namespace",
    "labels": {
      "env": "production",
      "team": "devops"
    },
    "annotations": {
      "description": "This is an example namespace for demonstration purposes"
    }
  }
}
```

返回给前端的是 JSON 数据需要前端转化为 YAML 格式:

```
{
    "code": 0,
    "success": true,
    "data": {
        "apiVersion": "v1",
        "kind": "Namespace",
        "metadata": {
            "name": "example-namespace",
            "labels": {
                "env": "production",
                "team": "devops"
            },
            "annotations": {
                "description": "This is an example namespace for demonstration purposes"
            }
        }
    }
}
```

多选删除的时候，前端传递如下参数：

```
{
  "group": "apps", //资源分组,如果不填就是核心资源
  "version": "v1", //版本号
  "kind": "Deployment",//类别
  "resources": [
    {
      "name": "nginx-test",//资源名
      "namespace": "test", //命名空间
      "isNamespaced": true //是否是命名空间资源，
    },
    {
      "name": "nginx-test1",
      "namespace": "test1",
      "isNamespaced": true
    }
  ]
}
```

接口的鉴权通过 Kubernetes 的 RBAC 机制实现。前端在调用接口时，需要在请求头中添加 `Authorization: {{token}}`，其中的 `token` 使用 Kubernetes 集群分发的令牌，无需在 `token` 前额外添加 `Bearer` 标识。

# 4.接口设计

## 删除资源实例

```
DELETE 10.120.1.58/olympus-portal/k8s/clusters/:cluster/apiResource
```

### 入参

- param

|   |   |   |
|---|---|---|
|参数|类型|说明|
|cluster|string|集群名字|

```
{
  "group": "apps",
  "version": "v1",
  "kind": "Deployment",
  "resources": [
    {
      "name": "my-deployment",
      "namespace": "default",
      "isNamespaced": true
    },
    {
      "name": "another-deployment",
      "namespace": "prod",
      "isNamespaced": true
    }
  ]
}
```

### 出参

```
{
    "code": 0,
    "success": true,
}
```

## **集群资源通用接口**

### **详情**

#### **请求**

核心组

```
GET 10.120.1.58/olympus-portal/k8s/clusters/:cluster/api/:version/:resourcetype/:name
```

非核心组

```
GET 10.120.1.58/olympus-portal/k8s/clusters/:cluster/apis/:grup/:version/:resourcetype/:name
```

#### **入参**

请求头

|   |   |   |
|---|---|---|
|参数|类型|说明|
|Authorization|string|用户token|
|Content-Type|string|application/json -> json 格式传输<br><br>application/yaml -> yaml 格式传输|

路径参数path

|   |   |   |
|---|---|---|
|参数|类型|说明|
|cluster|string|集群名称|
|namespace|string|命名空间|
|group|string|组类型 如 apps|
|version|string|资源版本 如 v1|
|resourcetype|string|资源类型 如 deployments|
|name|string|资源名称|

#### **出参**

对应的资源结构体

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "Deployment",
        "apiVersion": "apps/v1",
        "metadata": {
            "name": "olympus-core",
            "namespace": "caas-system",
            "uid": "d3139300-08c9-4188-b80a-217d7266a412",
            "resourceVersion": "14061468",
            "generation": 7,
            "creationTimestamp": "2024-11-06T18:37:08Z",
            "labels": {
                "app": "olympus-core",
                "app.kubernetes.io/managed-by": "Helm"
            },
            "annotations": {
                "deployment.kubernetes.io/revision": "7",
                "meta.helm.sh/release-name": "olympus-core",
                "meta.helm.sh/release-namespace": "caas-system"
            }
        },
        "spec": {
            "replicas": 1,
            "selector": {
                "matchLabels": {
                    "app": "olympus-core"
                }
            },
            "template": {
                "metadata": {
                    "creationTimestamp": null,
                    "labels": {
                        "app": "olympus-core"
                    }
                },
                "spec": {
                    "volumes": [
                        {
                            "name": "olympus-core-cm",
                            "configMap": {
                                "name": "olympus-core-cm",
                                "items": [
                                    {
                                        "key": "application.yml",
                                        "path": "application.yml"
                                    }
                                ],
                                "defaultMode": 420
                            }
                        },
                        {
                            "name": "docker",
                            "hostPath": {
                                "path": "/var/run",
                                "type": ""
                            }
                        },
                        {
                            "name": "logdir",
                            "emptyDir": {}
                        },
                        {
                            "name": "olympus-core-pvc",
                            "persistentVolumeClaim": {
                                "claimName": "olympus-core-pvc"
                            }
                        },
                        {
                            "name": "kube-api-access",
                            "secret": {
                                "secretName": "olympus-sa-secret",
                                "defaultMode": 420
                            }
                        }
                    ],
                    "containers": [
                        {
                            "name": "olympus-core",
                            "image": "10.10.103.155/k8s-deploy/olympus-core:v3.5.1-6dd0dd736",
                            "command": [
                                "java"
                            ],
                            "args": [
                                "-server",
                                "-XX:+UseContainerSupport",
                                "-XX:MaxRAMPercentage=75.0",
                                "-XX:InitialRAMPercentage=75.0",
                                "-XX:MinRAMPercentage=75.0",
                                "-jar",
                                "/usr/local/olympus-core.jar",
                                "--spring.config.location=/usr/local/conf/application.yml"
                            ],
                            "ports": [
                                {
                                    "name": "api",
                                    "containerPort": 8080,
                                    "protocol": "TCP"
                                }
                            ],
                            "env": [
                                {
                                    "name": "mysql_username",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "olympus-mysql-secret",
                                            "key": "username"
                                        }
                                    }
                                },
                                {
                                    "name": "mysql_password",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "olympus-mysql-secret",
                                            "key": "password"
                                        }
                                    }
                                },
                                {
                                    "name": "grafana_user",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "grafana-secret",
                                            "key": "user"
                                        }
                                    }
                                },
                                {
                                    "name": "grafana_password",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "grafana-secret",
                                            "key": "password"
                                        }
                                    }
                                },
                                {
                                    "name": "es_esName",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "elesticsearch-secret",
                                            "key": "esName"
                                        }
                                    }
                                },
                                {
                                    "name": "es_esPwd",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "elesticsearch-secret",
                                            "key": "esPwd"
                                        }
                                    }
                                },
                                {
                                    "name": "redis_password",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "olympus-redis-secret",
                                            "key": "password"
                                        }
                                    }
                                },
                                {
                                    "name": "aliyun_logs_logstash",
                                    "value": "/logs/*"
                                },
                                {
                                    "name": "aliyun_logs_logstash_tags",
                                    "value": "k8s_resource_type=Deployment,k8s_resource_name=olympus-core"
                                },
                                {
                                    "name": "TZ",
                                    "value": "Asia/Shanghai"
                                }
                            ],
                            "resources": {
                                "limits": {
                                    "cpu": "2",
                                    "memory": "4Gi"
                                },
                                "requests": {
                                    "cpu": "1",
                                    "memory": "2Gi"
                                }
                            },
                            "volumeMounts": [
                                {
                                    "name": "olympus-core-cm",
                                    "mountPath": "/usr/local/conf/application.yml",
                                    "subPath": "application.yml"
                                },
                                {
                                    "name": "docker",
                                    "mountPath": "/var/run"
                                },
                                {
                                    "name": "logdir",
                                    "mountPath": "/logs"
                                },
                                {
                                    "name": "olympus-core-pvc",
                                    "mountPath": "/usr/local/olympus-core"
                                },
                                {
                                    "name": "kube-api-access",
                                    "readOnly": true,
                                    "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
                                }
                            ],
                            "livenessProbe": {
                                "httpGet": {
                                    "path": "/openapi/healthy",
                                    "port": 8080,
                                    "scheme": "HTTP"
                                },
                                "initialDelaySeconds": 180,
                                "timeoutSeconds": 10,
                                "periodSeconds": 20,
                                "successThreshold": 1,
                                "failureThreshold": 3
                            },
                            "readinessProbe": {
                                "httpGet": {
                                    "path": "/openapi/healthy",
                                    "port": 8080,
                                    "scheme": "HTTP"
                                },
                                "initialDelaySeconds": 60,
                                "timeoutSeconds": 10,
                                "periodSeconds": 20,
                                "successThreshold": 1,
                                "failureThreshold": 5
                            },
                            "lifecycle": {
                                "postStart": {
                                    "exec": {
                                        "command": [
                                            "/bin/sh",
                                            "-c",
                                            "if [ ! -d '/usr/local/olympus-core/images' ]; then cp -r /usr/local/olympus-core-pv/images /usr/local/olympus-core/ ;fi && cp -r /usr/local/olympus-core-pv/doc /usr/local/olympus-core/ && cp /usr/local/olympus-core-pv/images/cluster/deploy-proxy.sh /usr/local/olympus-core/images/cluster/deploy-proxy.sh"
                                        ]
                                    }
                                }
                            },
                            "terminationMessagePath": "/dev/termination-log",
                            "terminationMessagePolicy": "File",
                            "imagePullPolicy": "Always"
                        }
                    ],
                    "restartPolicy": "Always",
                    "terminationGracePeriodSeconds": 30,
                    "dnsPolicy": "ClusterFirst",
                    "automountServiceAccountToken": false,
                    "securityContext": {},
                    "affinity": {
                        "podAntiAffinity": {
                            "preferredDuringSchedulingIgnoredDuringExecution": [
                                {
                                    "weight": 70,
                                    "podAffinityTerm": {
                                        "labelSelector": {
                                            "matchExpressions": [
                                                {
                                                    "key": "app",
                                                    "operator": "In",
                                                    "values": [
                                                        "olympus-core"
                                                    ]
                                                }
                                            ]
                                        },
                                        "topologyKey": "kubernetes.io/hostname"
                                    }
                                }
                            ]
                        }
                    },
                    "schedulerName": "default-scheduler"
                }
            },
            "strategy": {
                "type": "RollingUpdate",
                "rollingUpdate": {
                    "maxUnavailable": "25%",
                    "maxSurge": "25%"
                }
            },
            "revisionHistoryLimit": 10,
            "progressDeadlineSeconds": 600
        },
        "status": {
            "observedGeneration": 7,
            "replicas": 1,
            "updatedReplicas": 1,
            "readyReplicas": 1,
            "availableReplicas": 1,
            "conditions": [
                {
                    "type": "Available",
                    "status": "True",
                    "lastUpdateTime": "2024-11-11T09:17:04Z",
                    "lastTransitionTime": "2024-11-11T09:17:04Z",
                    "reason": "MinimumReplicasAvailable",
                    "message": "Deployment has minimum availability."
                },
                {
                    "type": "Progressing",
                    "status": "True",
                    "lastUpdateTime": "2024-11-11T10:06:06Z",
                    "lastTransitionTime": "2024-11-06T18:37:08Z",
                    "reason": "NewReplicaSetAvailable",
                    "message": "ReplicaSet \"olympus-core-99cc5b98c\" has successfully progressed."
                }
            ]
        }
    }
}
```

#### **例1: 获取命名空间(核心组)**

axios 代码

```
const axios = require('axios');

let config = {
  method: 'get',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/api/v1/namespaces/default',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  }
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "Namespace",
        "apiVersion": "v1",
        "metadata": {
            "name": "default",
            "uid": "60124ce6-34a4-4adf-bde1-b27b9494d52d",
            "resourceVersion": "1115",
            "creationTimestamp": "2024-11-06T02:56:00Z",
            "labels": {
                "kubernetes.io/metadata.name": "default",
                "skyview/nodepool": "default"
            },
            "annotations": {
                "skyview/nodepool": "default"
            },
            "finalizers": [
                "skyview/nodepool"
            ]
        },
        "spec": {
            "finalizers": [
                "kubernetes"
            ]
        },
        "status": {
            "phase": "Active"
        }
    }
}
```

#### **例2: 获取存储类(非核心组)**

axios 代码

```
const axios = require('axios');

let config = {
  method: 'get',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/apis/storage.k8s.io/v1/storageclasses/default',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  }
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "StorageClass",
        "apiVersion": "storage.k8s.io/v1",
        "metadata": {
            "name": "default",
            "uid": "04128550-25ee-4fba-8283-2f3c2ca26e1d",
            "resourceVersion": "4064",
            "creationTimestamp": "2024-11-06T03:06:55Z",
            "annotations": {
                "kubectl.kubernetes.io/last-applied-configuration": "{\"allowVolumeExpansion\":true,\"apiVersion\":\"storage.k8s.io/v1\",\"kind\":\"StorageClass\",\"metadata\":{\"name\":\"default\"},\"parameters\":{\"csi.storage.k8s.io/controller-expand-secret-name\":\"default\",\"csi.storage.k8s.io/controller-expand-secret-namespace\":\"kube-system\",\"csi.storage.k8s.io/node-publish-secret-name\":\"default\",\"csi.storage.k8s.io/node-publish-secret-namespace\":\"kube-system\",\"csi.storage.k8s.io/provisioner-secret-name\":\"default\",\"csi.storage.k8s.io/provisioner-secret-namespace\":\"kube-system\",\"driverName\":\"s3minio\"},\"provisioner\":\"object.csi.aliyun.com\"}"
            }
        },
        "provisioner": "object.csi.aliyun.com",
        "parameters": {
            "csi.storage.k8s.io/controller-expand-secret-name": "default",
            "csi.storage.k8s.io/controller-expand-secret-namespace": "kube-system",
            "csi.storage.k8s.io/node-publish-secret-name": "default",
            "csi.storage.k8s.io/node-publish-secret-namespace": "kube-system",
            "csi.storage.k8s.io/provisioner-secret-name": "default",
            "csi.storage.k8s.io/provisioner-secret-namespace": "kube-system",
            "driverName": "s3minio"
        },
        "reclaimPolicy": "Delete",
        "allowVolumeExpansion": true,
        "volumeBindingMode": "Immediate"
    }
}
```

### **创建**

#### **请求**

核心组

```
POST 10.120.1.58/olympus-portal/k8s/clusters/:cluster/api/:version/:resourcetype
```

非核心组

```
POST 10.120.1.58/olympus-portal/k8s/clusters/:cluster/apis/:grup/:version/:resourcetype
```

#### **入参**

请求头

|   |   |   |
|---|---|---|
|参数|类型|说明|
|Authorization|string|用户token|
|Content-Type|string|application/json -> json 格式传输<br><br>application/yaml -> yaml 格式传输|

路径参数path

|   |   |   |
|---|---|---|
|参数|类型|说明|
|cluster|string|集群名称|
|namespace|string|命名空间|
|group|string|组类型 如 apps|
|version|string|资源版本 如 v1|
|resourcetype|string|资源类型 如 deployments|

请求体body

content-type 为json

```
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "example-namespace",
    "labels": {
      "env": "production",
      "team": "devops"
    },
    "annotations": {
      "description": "This is an example namespace for demonstration purposes"
    }
  }
}
```

content-type 为yaml

```
apiVersion: v1
kind: Namespace
metadata:
  name: example-namespace
  labels:
    env: production
    team: devops
  annotations:
    description: "This is an example namespace for demonstration purposes"
```

#### **出参**

对应的资源结构体

```
{
    "code": 0,
    "success": true,
    "data": {
        "apiVersion": "v1",
        "kind": "Namespace",
        "metadata": {
            "name": "example-namespace",
            "labels": {
                "env": "production",
                "team": "devops"
            },
            "annotations": {
                "description": "This is an example namespace for demonstration purposes"
            }
        }
    }
}
```

#### **例1: 创建命名空间(核心组)**

axios 代码

```
const axios = require('axios');
let data = JSON.stringify({
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "hzhq-test",
    "labels": {
      "env": "test"
    },
    "annotations": {
      "description": "This is an example namespace"
    }
  }
});

let config = {
  method: 'post',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/api/v1/namespaces',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  },
  data : data
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "Namespace",
        "apiVersion": "v1",
        "metadata": {
            "name": "default",
            "uid": "60124ce6-34a4-4adf-bde1-b27b9494d52d",
            "resourceVersion": "1115",
            "creationTimestamp": "2024-11-06T02:56:00Z",
            "labels": {
                "kubernetes.io/metadata.name": "default",
                "skyview/nodepool": "default"
            },
            "annotations": {
                "skyview/nodepool": "default"
            },
            "finalizers": [
                "skyview/nodepool"
            ]
        },
        "spec": {
            "finalizers": [
                "kubernetes"
            ]
        },
        "status": {
            "phase": "Active"
        }
    }
}
```

#### **例2: 创建存储类(非核心组)**

axios

```
const axios = require('axios');
let data = JSON.stringify({
  "allowVolumeExpansion": false,
  "apiVersion": "storage.k8s.io/v1",
  "kind": "StorageClass",
  "metadata": {
    "annotations": {
      "storageservice.harmonycloud.cn/allocated": "0",
      "storageservice.harmonycloud.cn/capacity": "100",
      "storageservice.harmonycloud.cn/type": "NFS"
    },
    "creationTimestamp": "2024-11-07T02:24:43Z",
    "labels": {
      "a": "b"
    },
    "name": "hzhq-test"
  },
  "parameters": {
    "NFS_PATH": "/data/nfs",
    "NFS_SERVER": "10.10.103.155",
    "archiveOnDelete": "false"
  },
  "provisioner": "nfs-client-provisioner-by-nfs",
  "reclaimPolicy": "Delete",
  "volumeBindingMode": "Immediate"
});

let config = {
  method: 'post',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/apis/storage.k8s.io/v1/storageclasses',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  },
  data : data
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "metadata": {
            "name": "hzhq-test",
            "uid": "2e11743a-1332-4e5b-9aba-73281a8bab81",
            "resourceVersion": "32286427",
            "creationTimestamp": "2024-11-18T06:28:08Z",
            "labels": {
                "a": "b"
            },
            "annotations": {
                "storageservice.harmonycloud.cn/allocated": "0",
                "storageservice.harmonycloud.cn/capacity": "100",
                "storageservice.harmonycloud.cn/type": "NFS"
            },
            "managedFields": [
                {
                    "manager": "___run_server",
                    "operation": "Update",
                    "apiVersion": "storage.k8s.io/v1",
                    "time": "2024-11-18T06:28:08Z",
                    "fieldsType": "FieldsV1",
                    "fieldsV1": {
                        "f:allowVolumeExpansion": {},
                        "f:metadata": {
                            "f:annotations": {
                                ".": {},
                                "f:storageservice.harmonycloud.cn/allocated": {},
                                "f:storageservice.harmonycloud.cn/capacity": {},
                                "f:storageservice.harmonycloud.cn/type": {}
                            },
                            "f:labels": {
                                ".": {},
                                "f:a": {}
                            }
                        },
                        "f:parameters": {
                            ".": {},
                            "f:NFS_PATH": {},
                            "f:NFS_SERVER": {},
                            "f:archiveOnDelete": {}
                        },
                        "f:provisioner": {},
                        "f:reclaimPolicy": {},
                        "f:volumeBindingMode": {}
                    }
                }
            ]
        },
        "provisioner": "nfs-client-provisioner-by-nfs",
        "parameters": {
            "NFS_PATH": "/data/nfs",
            "NFS_SERVER": "10.10.103.155",
            "archiveOnDelete": "false"
        },
        "reclaimPolicy": "Delete",
        "allowVolumeExpansion": false,
        "volumeBindingMode": "Immediate"
    }
}
```

### **更新**

#### **请求**

核心组

```
PUT 10.120.1.58/olympus-portal/k8s/clusters/:cluster/api/:version/:resourcetype/:name
```

非核心组

```
PUT 10.120.1.58/olympus-portal/k8s/clusters/:cluster/apis/:grup/:version/:resourcetype/:name
```

#### **入参**

请求头

|   |   |   |
|---|---|---|
|参数|类型|说明|
|Authorization|string|用户token|
|Content-Type|string|application/json -> json 格式传输<br><br>application/yaml -> yaml 格式传输|

路径参数path

|   |   |   |
|---|---|---|
|参数|类型|说明|
|cluster|string|集群名称|
|namespace|string|命名空间|
|group|string|组类型 如 apps|
|version|string|资源版本 如 v1|
|resourcetype|string|资源类型 如 deployments|
|name|string|资源名称|

请求体body

content-type 为json

```
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "example-namespace",
    "labels": {
      "env": "production",
      "team": "devops"
    },
    "annotations": {
      "description": "This is an example namespace for demonstration purposes"
    }
  }
}
```

content-type 为yaml

```
apiVersion: v1
kind: Namespace
metadata:
  name: example-namespace
  labels:
    env: production
    team: devops
  annotations:
    description: "This is an example namespace for demonstration purposes"
```

#### **出参**

对应的更改后的资源结构体

```
{
    "code": 0,
    "success": true,
    "data": {
        "apiVersion": "v1",
        "kind": "Namespace",
        "metadata": {
            "name": "example-namespace",
            "labels": {
                "env": "production",
                "team": "devops"
            },
            "annotations": {
                "description": "This is an example namespace for demonstration purposes"
            }
        }
    }
}
```

#### **例1: 更新命名空间(核心组)**

axios 代码

命名空间无法被修改。

```
const axios = require('axios');
let data = JSON.stringify({
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "hzhq-test",
    "labels": {
      "env": "test",
      "a": "b"
    },
    "annotations": {
      "description": "This is an example namespace"
    }
  }
});

let config = {
  method: 'put',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/api/v1/namespaces/hzhq-test',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  },
  data : data
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 10006,
    "success": false,
    "errorMsg": "参数错误",
    "errorDetail": "object namespace could not be change"
}
```

#### **例2: 更新存储类(非核心组)**

axios 代码

```
const axios = require('axios');
let data = JSON.stringify({
  "allowVolumeExpansion": false,
  "apiVersion": "storage.k8s.io/v1",
  "kind": "StorageClass",
  "metadata": {
    "annotations": {
      "storageservice.harmonycloud.cn/allocated": "0",
      "storageservice.harmonycloud.cn/capacity": "100",
      "storageservice.harmonycloud.cn/type": "NFS"
    },
    "creationTimestamp": "2024-11-07T02:24:43Z",
    "labels": {
      "a": "b",
      "c": "d"
    },
    "name": "hzhq-test"
  },
  "parameters": {
    "NFS_PATH": "/data/nfs",
    "NFS_SERVER": "10.10.103.155",
    "archiveOnDelete": "false"
  },
  "provisioner": "nfs-client-provisioner-by-nfs",
  "reclaimPolicy": "Delete",
  "volumeBindingMode": "Immediate"
});

let config = {
  method: 'put',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/apis/storage.k8s.io/v1/storageclasses/hzhq-test',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  },
  data : data
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "StorageClass",
        "apiVersion": "storage.k8s.io/v1",
        "metadata": {
            "name": "hzhq-test",
            "uid": "2e11743a-1332-4e5b-9aba-73281a8bab81",
            "resourceVersion": "32297188",
            "creationTimestamp": "2024-11-18T06:28:08Z",
            "labels": {
                "a": "b",
                "c": "d"
            },
            "annotations": {
                "storageservice.harmonycloud.cn/allocated": "0",
                "storageservice.harmonycloud.cn/capacity": "100",
                "storageservice.harmonycloud.cn/type": "NFS"
            },
            "managedFields": [
                {
                    "manager": "___run_server",
                    "operation": "Update",
                    "apiVersion": "storage.k8s.io/v1",
                    "time": "2024-11-18T06:33:20Z",
                    "fieldsType": "FieldsV1",
                    "fieldsV1": {
                        "f:allowVolumeExpansion": {},
                        "f:metadata": {
                            "f:annotations": {
                                ".": {},
                                "f:storageservice.harmonycloud.cn/allocated": {},
                                "f:storageservice.harmonycloud.cn/capacity": {},
                                "f:storageservice.harmonycloud.cn/type": {}
                            },
                            "f:labels": {
                                ".": {},
                                "f:a": {},
                                "f:c": {}
                            }
                        },
                        "f:parameters": {
                            ".": {},
                            "f:NFS_PATH": {},
                            "f:NFS_SERVER": {},
                            "f:archiveOnDelete": {}
                        },
                        "f:provisioner": {},
                        "f:reclaimPolicy": {},
                        "f:volumeBindingMode": {}
                    }
                }
            ]
        },
        "provisioner": "nfs-client-provisioner-by-nfs",
        "parameters": {
            "NFS_PATH": "/data/nfs",
            "NFS_SERVER": "10.10.103.155",
            "archiveOnDelete": "false"
        },
        "reclaimPolicy": "Delete",
        "allowVolumeExpansion": false,
        "volumeBindingMode": "Immediate"
    }
}
```

## **命名空间资源通用接口**

### **详情**

#### **请求**

核心组

```
GET 10.120.1.58/olympus-portal/k8s/clusters/:cluster/api/:version/namespaces/:namespace/:resourcetype/:name
```

非核心组

```
GET 10.120.1.58/olympus-portal/k8s/clusters/:cluster/apis/:grup/:version/:resourcetype/:name
```

#### **入参**

请求头

|   |   |   |
|---|---|---|
|参数|类型|说明|
|Authorization|string|用户token|
|Content-Type|string|application/json -> json 格式传输<br><br>application/yaml -> yaml 格式传输|

路径参数path

|   |   |   |
|---|---|---|
|参数|类型|说明|
|cluster|string|集群名称|
|namespace|string|命名空间|
|group|string|组类型 如 apps|
|version|string|资源版本 如 v1|
|resourcetype|string|资源类型 如 deployments|
|name|string|资源名称|

#### **出参**

对应的资源结构体

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "Deployment",
        "apiVersion": "apps/v1",
        "metadata": {
            "name": "olympus-core",
            "namespace": "caas-system",
            "uid": "d3139300-08c9-4188-b80a-217d7266a412",
            "resourceVersion": "14061468",
            "generation": 7,
            "creationTimestamp": "2024-11-06T18:37:08Z",
            "labels": {
                "app": "olympus-core",
                "app.kubernetes.io/managed-by": "Helm"
            },
            "annotations": {
                "deployment.kubernetes.io/revision": "7",
                "meta.helm.sh/release-name": "olympus-core",
                "meta.helm.sh/release-namespace": "caas-system"
            }
        },
        "spec": {
            "replicas": 1,
            "selector": {
                "matchLabels": {
                    "app": "olympus-core"
                }
            },
            "template": {
                "metadata": {
                    "creationTimestamp": null,
                    "labels": {
                        "app": "olympus-core"
                    }
                },
                "spec": {
                    "volumes": [
                        {
                            "name": "olympus-core-cm",
                            "configMap": {
                                "name": "olympus-core-cm",
                                "items": [
                                    {
                                        "key": "application.yml",
                                        "path": "application.yml"
                                    }
                                ],
                                "defaultMode": 420
                            }
                        },
                        {
                            "name": "docker",
                            "hostPath": {
                                "path": "/var/run",
                                "type": ""
                            }
                        },
                        {
                            "name": "logdir",
                            "emptyDir": {}
                        },
                        {
                            "name": "olympus-core-pvc",
                            "persistentVolumeClaim": {
                                "claimName": "olympus-core-pvc"
                            }
                        },
                        {
                            "name": "kube-api-access",
                            "secret": {
                                "secretName": "olympus-sa-secret",
                                "defaultMode": 420
                            }
                        }
                    ],
                    "containers": [
                        {
                            "name": "olympus-core",
                            "image": "10.10.103.155/k8s-deploy/olympus-core:v3.5.1-6dd0dd736",
                            "command": [
                                "java"
                            ],
                            "args": [
                                "-server",
                                "-XX:+UseContainerSupport",
                                "-XX:MaxRAMPercentage=75.0",
                                "-XX:InitialRAMPercentage=75.0",
                                "-XX:MinRAMPercentage=75.0",
                                "-jar",
                                "/usr/local/olympus-core.jar",
                                "--spring.config.location=/usr/local/conf/application.yml"
                            ],
                            "ports": [
                                {
                                    "name": "api",
                                    "containerPort": 8080,
                                    "protocol": "TCP"
                                }
                            ],
                            "env": [
                                {
                                    "name": "mysql_username",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "olympus-mysql-secret",
                                            "key": "username"
                                        }
                                    }
                                },
                                {
                                    "name": "mysql_password",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "olympus-mysql-secret",
                                            "key": "password"
                                        }
                                    }
                                },
                                {
                                    "name": "grafana_user",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "grafana-secret",
                                            "key": "user"
                                        }
                                    }
                                },
                                {
                                    "name": "grafana_password",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "grafana-secret",
                                            "key": "password"
                                        }
                                    }
                                },
                                {
                                    "name": "es_esName",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "elesticsearch-secret",
                                            "key": "esName"
                                        }
                                    }
                                },
                                {
                                    "name": "es_esPwd",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "elesticsearch-secret",
                                            "key": "esPwd"
                                        }
                                    }
                                },
                                {
                                    "name": "redis_password",
                                    "valueFrom": {
                                        "secretKeyRef": {
                                            "name": "olympus-redis-secret",
                                            "key": "password"
                                        }
                                    }
                                },
                                {
                                    "name": "aliyun_logs_logstash",
                                    "value": "/logs/*"
                                },
                                {
                                    "name": "aliyun_logs_logstash_tags",
                                    "value": "k8s_resource_type=Deployment,k8s_resource_name=olympus-core"
                                },
                                {
                                    "name": "TZ",
                                    "value": "Asia/Shanghai"
                                }
                            ],
                            "resources": {
                                "limits": {
                                    "cpu": "2",
                                    "memory": "4Gi"
                                },
                                "requests": {
                                    "cpu": "1",
                                    "memory": "2Gi"
                                }
                            },
                            "volumeMounts": [
                                {
                                    "name": "olympus-core-cm",
                                    "mountPath": "/usr/local/conf/application.yml",
                                    "subPath": "application.yml"
                                },
                                {
                                    "name": "docker",
                                    "mountPath": "/var/run"
                                },
                                {
                                    "name": "logdir",
                                    "mountPath": "/logs"
                                },
                                {
                                    "name": "olympus-core-pvc",
                                    "mountPath": "/usr/local/olympus-core"
                                },
                                {
                                    "name": "kube-api-access",
                                    "readOnly": true,
                                    "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
                                }
                            ],
                            "livenessProbe": {
                                "httpGet": {
                                    "path": "/openapi/healthy",
                                    "port": 8080,
                                    "scheme": "HTTP"
                                },
                                "initialDelaySeconds": 180,
                                "timeoutSeconds": 10,
                                "periodSeconds": 20,
                                "successThreshold": 1,
                                "failureThreshold": 3
                            },
                            "readinessProbe": {
                                "httpGet": {
                                    "path": "/openapi/healthy",
                                    "port": 8080,
                                    "scheme": "HTTP"
                                },
                                "initialDelaySeconds": 60,
                                "timeoutSeconds": 10,
                                "periodSeconds": 20,
                                "successThreshold": 1,
                                "failureThreshold": 5
                            },
                            "lifecycle": {
                                "postStart": {
                                    "exec": {
                                        "command": [
                                            "/bin/sh",
                                            "-c",
                                            "if [ ! -d '/usr/local/olympus-core/images' ]; then cp -r /usr/local/olympus-core-pv/images /usr/local/olympus-core/ ;fi && cp -r /usr/local/olympus-core-pv/doc /usr/local/olympus-core/ && cp /usr/local/olympus-core-pv/images/cluster/deploy-proxy.sh /usr/local/olympus-core/images/cluster/deploy-proxy.sh"
                                        ]
                                    }
                                }
                            },
                            "terminationMessagePath": "/dev/termination-log",
                            "terminationMessagePolicy": "File",
                            "imagePullPolicy": "Always"
                        }
                    ],
                    "restartPolicy": "Always",
                    "terminationGracePeriodSeconds": 30,
                    "dnsPolicy": "ClusterFirst",
                    "automountServiceAccountToken": false,
                    "securityContext": {},
                    "affinity": {
                        "podAntiAffinity": {
                            "preferredDuringSchedulingIgnoredDuringExecution": [
                                {
                                    "weight": 70,
                                    "podAffinityTerm": {
                                        "labelSelector": {
                                            "matchExpressions": [
                                                {
                                                    "key": "app",
                                                    "operator": "In",
                                                    "values": [
                                                        "olympus-core"
                                                    ]
                                                }
                                            ]
                                        },
                                        "topologyKey": "kubernetes.io/hostname"
                                    }
                                }
                            ]
                        }
                    },
                    "schedulerName": "default-scheduler"
                }
            },
            "strategy": {
                "type": "RollingUpdate",
                "rollingUpdate": {
                    "maxUnavailable": "25%",
                    "maxSurge": "25%"
                }
            },
            "revisionHistoryLimit": 10,
            "progressDeadlineSeconds": 600
        },
        "status": {
            "observedGeneration": 7,
            "replicas": 1,
            "updatedReplicas": 1,
            "readyReplicas": 1,
            "availableReplicas": 1,
            "conditions": [
                {
                    "type": "Available",
                    "status": "True",
                    "lastUpdateTime": "2024-11-11T09:17:04Z",
                    "lastTransitionTime": "2024-11-11T09:17:04Z",
                    "reason": "MinimumReplicasAvailable",
                    "message": "Deployment has minimum availability."
                },
                {
                    "type": "Progressing",
                    "status": "True",
                    "lastUpdateTime": "2024-11-11T10:06:06Z",
                    "lastTransitionTime": "2024-11-06T18:37:08Z",
                    "reason": "NewReplicaSetAvailable",
                    "message": "ReplicaSet \"olympus-core-99cc5b98c\" has successfully progressed."
                }
            ]
        }
    }
}
```

#### **例1: 获取配置文件(核心组)**

axios 代码

```
const axios = require('axios');

let config = {
  method: 'get',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/api/v1/namespaces/default/configmaps/hzhq-test',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  }
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "ConfigMap",
        "apiVersion": "v1",
        "metadata": {
            "name": "hzhq-test",
            "namespace": "default",
            "uid": "aee582c7-eefe-47fd-9161-b95ed19d5bdb",
            "resourceVersion": "32375285",
            "creationTimestamp": "2024-11-18T07:13:37Z"
        }
    }
}
```

#### **例2: 获取无状态部署(非核心组)**

axios 代码

```
const axios = require('axios');

let config = {
  method: 'get',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/apis/apps/v1/namespaces/default/deployments/appv1',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  }
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "Deployment",
        "apiVersion": "apps/v1",
        "metadata": {
            "name": "appv1",
            "namespace": "default",
            "uid": "8708ac54-e945-4dd8-afe6-2c5667e3c938",
            "resourceVersion": "4612494",
            "generation": 2,
            "creationTimestamp": "2024-11-07T02:16:16Z",
            "labels": {
                "app": "v1"
            },
            "annotations": {
                "deployment.kubernetes.io/revision": "2",
                "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"apps/v1\",\"kind\":\"Deployment\",\"metadata\":{\"annotations\":{},\"labels\":{\"app\":\"v1\"},\"name\":\"appv1\",\"namespace\":\"default\"},\"spec\":{\"replicas\":1,\"selector\":{\"matchLabels\":{\"app\":\"v1\"}},\"template\":{\"metadata\":{\"labels\":{\"app\":\"v1\"}},\"spec\":{\"containers\":[{\"image\":\"10.10.103.155/library/tomcat:9\",\"name\":\"tomcat\",\"ports\":[{\"containerPort\":80}]}],\"tolerations\":[{\"operator\":\"Exists\"}]}}}}\n"
            }
        },
        "spec": {
            "replicas": 1,
            "selector": {
                "matchLabels": {
                    "app": "v1"
                }
            },
            "template": {
                "metadata": {
                    "creationTimestamp": null,
                    "labels": {
                        "app": "v1"
                    }
                },
                "spec": {
                    "containers": [
                        {
                            "name": "tomcat",
                            "image": "10.10.103.155/library/tomcat:9",
                            "ports": [
                                {
                                    "containerPort": 8080,
                                    "protocol": "TCP"
                                }
                            ],
                            "resources": {},
                            "terminationMessagePath": "/dev/termination-log",
                            "terminationMessagePolicy": "File",
                            "imagePullPolicy": "IfNotPresent"
                        }
                    ],
                    "restartPolicy": "Always",
                    "terminationGracePeriodSeconds": 30,
                    "dnsPolicy": "ClusterFirst",
                    "securityContext": {},
                    "schedulerName": "default-scheduler",
                    "tolerations": [
                        {
                            "operator": "Exists"
                        }
                    ]
                }
            },
            "strategy": {
                "type": "RollingUpdate",
                "rollingUpdate": {
                    "maxUnavailable": "25%",
                    "maxSurge": "25%"
                }
            },
            "revisionHistoryLimit": 10,
            "progressDeadlineSeconds": 600
        },
        "status": {
            "observedGeneration": 2,
            "replicas": 1,
            "updatedReplicas": 1,
            "readyReplicas": 1,
            "availableReplicas": 1,
            "conditions": [
                {
                    "type": "Available",
                    "status": "True",
                    "lastUpdateTime": "2024-11-07T02:52:27Z",
                    "lastTransitionTime": "2024-11-07T02:52:27Z",
                    "reason": "MinimumReplicasAvailable",
                    "message": "Deployment has minimum availability."
                },
                {
                    "type": "Progressing",
                    "status": "True",
                    "lastUpdateTime": "2024-11-08T02:39:40Z",
                    "lastTransitionTime": "2024-11-07T02:52:07Z",
                    "reason": "NewReplicaSetAvailable",
                    "message": "ReplicaSet \"appv1-7d467c749f\" has successfully progressed."
                }
            ]
        }
    }
}
```

### **创建**

#### **请求**

核心组

```
POST 10.120.1.58/olympus-portal/k8s/clusters/:cluster/api/:version/namespaces/:namespace/:resourcetype
```

非核心组

```
POST 10.120.1.58/olympus-portal/k8s/clusters/:cluster/apis/:grup/:version/namespaces/:namespace/:resourcetype
```

#### **入参**

请求头

|   |   |   |
|---|---|---|
|参数|类型|说明|
|Authorization|string|用户token|
|Content-Type|string|application/json -> json 格式传输<br><br>application/yaml -> yaml 格式传输|

路径参数path

|   |   |   |
|---|---|---|
|参数|类型|说明|
|cluster|string|集群名称|
|namespace|string|命名空间|
|group|string|组类型 如 apps|
|version|string|资源版本 如 v1|
|resourcetype|string|资源类型 如 deployments|

请求体body

content-type 为json

```
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "example-namespace",
    "labels": {
      "env": "production",
      "team": "devops"
    },
    "annotations": {
      "description": "This is an example namespace for demonstration purposes"
    }
  }
}
```

content-type 为yaml

```
apiVersion: v1
kind: Namespace
metadata:
  name: example-namespace
  labels:
    env: production
    team: devops
  annotations:
    description: "This is an example namespace for demonstration purposes"
```

#### **出参**

对应的资源结构体

```
{
    "code": 0,
    "success": true,
    "data": {
        "apiVersion": "v1",
        "kind": "Namespace",
        "metadata": {
            "name": "example-namespace",
            "labels": {
                "env": "production",
                "team": "devops"
            },
            "annotations": {
                "description": "This is an example namespace for demonstration purposes"
            }
        }
    }
}
```

#### **例1: 创建配置文件(核心组)**

axios 代码

```
const axios = require('axios');
let data = JSON.stringify({
  "apiVersion": "v1",
  "kind": "ConfigMap",
  "metadata": {
    "name": "hzhq-test",
    "labels": {
      "env": "test"
    },
    "annotations": {
      "description": "This is an example namespace"
    }
  }
});

let config = {
  method: 'post',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/api/v1/namespaces/default/configmaps',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  },
  data : data
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "apiVersion": "v1",
        "kind": "ConfigMap",
        "metadata": {
            "name": "hzhq-test",
            "labels": {
                "env": "test"
            },
            "annotations": {
                "description": "This is an example namespace"
            }
        }
    }
}
```

#### **例2: 创建无状态部署(非核心组)**

```
const axios = require('axios');
let data = JSON.stringify({
  "kind": "Deployment",
  "apiVersion": "apps/v1",
  "metadata": {
    "name": "appv1",
    "namespace": "default",
    "labels": {
      "app": "v1"
    }
  },
  "spec": {
    "replicas": 1,
    "selector": {
      "matchLabels": {
        "app": "v1"
      }
    },
    "template": {
      "metadata": {
        "creationTimestamp": null,
        "labels": {
          "app": "v1"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "tomcat",
            "image": "10.10.103.155/library/tomcat:9",
            "ports": [
              {
                "containerPort": 8080,
                "protocol": "TCP"
              }
            ],
            "resources": {},
            "terminationMessagePath": "/dev/termination-log",
            "terminationMessagePolicy": "File",
            "imagePullPolicy": "IfNotPresent"
          }
        ],
        "restartPolicy": "Always",
        "terminationGracePeriodSeconds": 30,
        "dnsPolicy": "ClusterFirst",
        "securityContext": {},
        "schedulerName": "default-scheduler",
        "tolerations": [
          {
            "operator": "Exists"
          }
        ]
      }
    },
    "strategy": {
      "type": "RollingUpdate",
      "rollingUpdate": {
        "maxUnavailable": "25%",
        "maxSurge": "25%"
      }
    },
    "revisionHistoryLimit": 10,
    "progressDeadlineSeconds": 600
  }
});

let config = {
  method: 'post',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/apis/apps/v1/namespaces/default/deployments',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  },
  data : data
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "Deployment",
        "apiVersion": "apps/v1",
        "metadata": {
            "name": "appv1",
            "namespace": "default",
            "labels": {
                "app": "v1"
            }
        },
        "spec": {
            "replicas": 1,
            "selector": {
                "matchLabels": {
                    "app": "v1"
                }
            },
            "template": {
                "metadata": {
                    "creationTimestamp": null,
                    "labels": {
                        "app": "v1"
                    }
                },
                "spec": {
                    "containers": [
                        {
                            "name": "tomcat",
                            "image": "10.10.103.155/library/tomcat:9",
                            "ports": [
                                {
                                    "containerPort": 8080,
                                    "protocol": "TCP"
                                }
                            ],
                            "resources": {},
                            "terminationMessagePath": "/dev/termination-log",
                            "terminationMessagePolicy": "File",
                            "imagePullPolicy": "IfNotPresent"
                        }
                    ],
                    "restartPolicy": "Always",
                    "terminationGracePeriodSeconds": 30,
                    "dnsPolicy": "ClusterFirst",
                    "securityContext": {},
                    "schedulerName": "default-scheduler",
                    "tolerations": [
                        {
                            "operator": "Exists"
                        }
                    ]
                }
            },
            "strategy": {
                "type": "RollingUpdate",
                "rollingUpdate": {
                    "maxUnavailable": "25%",
                    "maxSurge": "25%"
                }
            },
            "revisionHistoryLimit": 10,
            "progressDeadlineSeconds": 600
        }
    }
}
```

### **更新**

#### **请求**

核心组

```
PUT 10.120.1.58/olympus-portal/k8s/clusters/:cluster/api/:version/namespaces/:namespace/:resourcetype/:name
```

非核心组

```
PUT 10.120.1.58/olympus-portal/k8s/clusters/:cluster/apis/:grup/:version/namespaces/:namespace/:resourcetype/:name
```

#### **入参**

请求头

|   |   |   |
|---|---|---|
|参数|类型|说明|
|Authorization|string|用户token|
|Content-Type|string|application/json -> json 格式传输<br><br>application/yaml -> yaml 格式传输|

路径参数path

|   |   |   |
|---|---|---|
|参数|类型|说明|
|cluster|string|集群名称|
|namespace|string|命名空间|
|group|string|组类型 如 apps|
|version|string|资源版本 如 v1|
|resourcetype|string|资源类型 如 deployments|
|name|string|资源名称|

请求体body

content-type 为json

```
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "example-namespace",
    "labels": {
      "env": "production",
      "team": "devops"
    },
    "annotations": {
      "description": "This is an example namespace for demonstration purposes"
    }
  }
}
```

content-type 为yaml

```
apiVersion: v1
kind: Namespace
metadata:
  name: example-namespace
  labels:
    env: production
    team: devops
  annotations:
    description: "This is an example namespace for demonstration purposes"
```

#### **出参**

对应的更改后的资源结构体

```
{
    "code": 0,
    "success": true,
    "data": {
        "apiVersion": "v1",
        "kind": "Namespace",
        "metadata": {
            "name": "example-namespace",
            "labels": {
                "env": "production",
                "team": "devops"
            },
            "annotations": {
                "description": "This is an example namespace for demonstration purposes"
            }
        }
    }
}
```

#### **例1: 更新配置文件(核心组)**

axios 代码

```
const axios = require('axios');
let data = JSON.stringify({
  "apiVersion": "v1",
  "kind": "ConfigMap",
  "metadata": {
    "name": "hzhq-test",
    "labels": {
      "env": "test"
    },
    "annotations": {
      "description": "This is an example namespace"
    }
  }
});

let config = {
  method: 'put',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/api/v1/namespaces/default/configmaps',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  },
  data : data
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "apiVersion": "v1",
        "kind": "ConfigMap",
        "metadata": {
            "name": "hzhq-test",
            "labels": {
                "env": "test"
            },
            "annotations": {
                "description": "This is an example namespace"
            }
        }
    }
}
```

#### **例2: 更新无状态部署(非核心组)**

```
const axios = require('axios');
let data = JSON.stringify({
  "kind": "Deployment",
  "apiVersion": "apps/v1",
  "metadata": {
    "name": "appv1",
    "namespace": "default",
    "labels": {
      "app": "v1"
    }
  },
  "spec": {
    "replicas": 1,
    "selector": {
      "matchLabels": {
        "app": "v1"
      }
    },
    "template": {
      "metadata": {
        "creationTimestamp": null,
        "labels": {
          "app": "v1"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "tomcat",
            "image": "10.10.103.155/library/tomcat:9",
            "ports": [
              {
                "containerPort": 8080,
                "protocol": "TCP"
              }
            ],
            "resources": {},
            "terminationMessagePath": "/dev/termination-log",
            "terminationMessagePolicy": "File",
            "imagePullPolicy": "IfNotPresent"
          }
        ],
        "restartPolicy": "Always",
        "terminationGracePeriodSeconds": 30,
        "dnsPolicy": "ClusterFirst",
        "securityContext": {},
        "schedulerName": "default-scheduler",
        "tolerations": [
          {
            "operator": "Exists"
          }
        ]
      }
    },
    "strategy": {
      "type": "RollingUpdate",
      "rollingUpdate": {
        "maxUnavailable": "25%",
        "maxSurge": "25%"
      }
    },
    "revisionHistoryLimit": 10,
    "progressDeadlineSeconds": 600
  }
});

let config = {
  method: 'put',
  maxBodyLength: Infinity,
  url: 'http://localhost:8080/k8s/clusters/skyview-dev/apis/apps/v1/namespaces/default/namespaces',
  headers: { 
    'Content-Type': 'application/json', 
    'Authorization': 'eyJhbGciOiJSUzI1NiIsImtpZCI6ImxiN0xqVkV5cUVib3FQR3RJalFMM3pvZ3hPNHV2T0sxTTRyVkZ5NFUzNW8ifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNzYzMDgyMTM5LCJpYXQiOjE3MzE1NDYxMzksImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJzdGVsbGFyaXMtc3lzdGVtIiwicG9kIjp7Im5hbWUiOiJzdGVsbGFyaXMtcHJveHktcHJveHktNTdkOGRkZjhjNy1sNW54NyIsInVpZCI6Ijc2M2Y1MWMyLTkxYmQtNGM2MC1iZWFmLTcwOThkOWQ0MDNiMCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoic3RlbGxhcmlzLXByb3h5LXByb3h5IiwidWlkIjoiYTlhZWQxOTQtM2ZhNi00OTg2LWI0N2UtNTRhZGJkYWYzYTZkIn0sIndhcm5hZnRlciI6MTczMTU0OTc0Nn0sIm5iZiI6MTczMTU0NjEzOSwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnN0ZWxsYXJpcy1zeXN0ZW06c3RlbGxhcmlzLXByb3h5LXByb3h5In0.hvFcoWo7GDSS4M2AniB7K74kNH_9LWzTJYDakN68bNoi_6KjKF4MraZV5rUYI8xFiQQCf7h8oJ2rNT9FeoK92i4_qiNOOZSPhH_NTku3ieWFfDYysowT9MBgUN3KF6sY9fuwasGiIU7OQMMThkNbSQSWCKuaoZhW7N4Y7j2-4-472kW8NG05m_EPIHuuzoU-GtWQLccwUwKL7N_5p2jIcygPwSLbPZsDkyerNLvyUsp-WfGQtScasUZd6CP7Pm8-wid7ptjjHcQnSMxgz9y6OfjxZT6S8Pi-nYKu7-e-JvRFBAs0FND-s6bzzrfC1cTgU8oErRIGaFx9k0pL1dImbA'
  },
  data : data
};

axios.request(config)
.then((response) => {
  console.log(JSON.stringify(response.data));
})
.catch((error) => {
  console.log(error);
});
```

响应

```
{
    "code": 0,
    "success": true,
    "data": {
        "kind": "Deployment",
        "apiVersion": "apps/v1",
        "metadata": {
            "name": "appv1",
            "namespace": "default",
            "uid": "8708ac54-e945-4dd8-afe6-2c5667e3c938",
            "resourceVersion": "4612494",
            "generation": 2,
            "creationTimestamp": "2024-11-07T02:16:16Z",
            "labels": {
                "app": "v1"
            },
            "annotations": {
                "deployment.kubernetes.io/revision": "2",
                "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"apps/v1\",\"kind\":\"Deployment\",\"metadata\":{\"annotations\":{},\"labels\":{\"app\":\"v1\"},\"name\":\"appv1\",\"namespace\":\"default\"},\"spec\":{\"replicas\":1,\"selector\":{\"matchLabels\":{\"app\":\"v1\"}},\"template\":{\"metadata\":{\"labels\":{\"app\":\"v1\"}},\"spec\":{\"containers\":[{\"image\":\"10.10.103.155/library/tomcat:9\",\"name\":\"tomcat\",\"ports\":[{\"containerPort\":80}]}],\"tolerations\":[{\"operator\":\"Exists\"}]}}}}\n"
            }
        },
        "spec": {
            "replicas": 1,
            "selector": {
                "matchLabels": {
                    "app": "v1"
                }
            },
            "template": {
                "metadata": {
                    "creationTimestamp": null,
                    "labels": {
                        "app": "v1"
                    }
                },
                "spec": {
                    "containers": [
                        {
                            "name": "tomcat",
                            "image": "10.10.103.155/library/tomcat:9",
                            "ports": [
                                {
                                    "containerPort": 8080,
                                    "protocol": "TCP"
                                }
                            ],
                            "resources": {},
                            "terminationMessagePath": "/dev/termination-log",
                            "terminationMessagePolicy": "File",
                            "imagePullPolicy": "IfNotPresent"
                        }
                    ],
                    "restartPolicy": "Always",
                    "terminationGracePeriodSeconds": 30,
                    "dnsPolicy": "ClusterFirst",
                    "securityContext": {},
                    "schedulerName": "default-scheduler",
                    "tolerations": [
                        {
                            "operator": "Exists"
                        }
                    ]
                }
            },
            "strategy": {
                "type": "RollingUpdate",
                "rollingUpdate": {
                    "maxUnavailable": "25%",
                    "maxSurge": "25%"
                }
            },
            "revisionHistoryLimit": 10,
            "progressDeadlineSeconds": 600
        },
        "status": {
            "observedGeneration": 2,
            "replicas": 1,
            "updatedReplicas": 1,
            "readyReplicas": 1,
            "availableReplicas": 1,
            "conditions": [
                {
                    "type": "Available",
                    "status": "True",
                    "lastUpdateTime": "2024-11-07T02:52:27Z",
                    "lastTransitionTime": "2024-11-07T02:52:27Z",
                    "reason": "MinimumReplicasAvailable",
                    "message": "Deployment has minimum availability."
                },
                {
                    "type": "Progressing",
                    "status": "True",
                    "lastUpdateTime": "2024-11-08T02:39:40Z",
                    "lastTransitionTime": "2024-11-07T02:52:07Z",
                    "reason": "NewReplicaSetAvailable",
                    "message": "ReplicaSet \"appv1-7d467c749f\" has successfully progressed."
                }
            ]
        }
    }
}
```