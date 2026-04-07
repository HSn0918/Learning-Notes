#elasticsearch #docker

相关笔记：[[elasticsearch-basics]] | [[es-field-types]]

## ES + Kibana Docker 部署

本文介绍使用 Docker Compose 快速搭建 Elasticsearch + Kibana 开发环境。

### 架构概览

```mermaid
graph LR
    Client["浏览器 / API 客户端"] -->|"9200"| ES["Elasticsearch<br/>8.6.0"]
    Client -->|"5601"| Kibana["Kibana<br/>8.6.0"]
    Kibana -->|内部通信| ES
    ES -->|挂载| Plugins["./plugins 目录"]

    subgraph "Docker Network: elastic"
        ES
        Kibana
    end
```

## 部署步骤

### Step 1：创建 Docker 网络

```bash
docker network create elastic
```

### Step 2：docker-compose.yml（Elasticsearch）

```yaml
version: '3'
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.6.0
    container_name: elasticsearch
    networks:
      - elastic
    ports:
      - 9200:9200
    environment:
      - discovery.type=single-node
      - ES_JAVA_OPTS=-Xms1g -Xmx1g
      - xpack.security.enabled=false
    volumes:
      - ./plugins:/usr/share/elasticsearch/plugins
    stdin_open: true
    tty: true

networks:
  elastic:
    external: true
```

关键配置说明：

| 配置项 | 说明 |
|-------|------|
| `discovery.type=single-node` | 单节点模式，适合开发环境 |
| `ES_JAVA_OPTS=-Xms1g -Xmx1g` | JVM 堆内存设为 1GB，可根据机器配置调整 |
| `xpack.security.enabled=false` | 关闭安全认证，仅限开发环境使用 |
| `./plugins` 挂载 | 方便安装 IK 分词器等插件 |

### Step 3：启动 Elasticsearch

```bash
docker-compose up -d
```

### Step 4：启动 Kibana

```bash
docker run --name kibana \
  --net elastic \
  -p 5601:5601 \
  docker.elastic.co/kibana/kibana:8.6.0
```

> Kibana 版本需要与 Elasticsearch 版本一致。

### Step 5：验证部署

```bash
# 检查 ES 是否正常运行
curl http://localhost:9200

# 预期返回集群信息 JSON
# {
#   "name": "...",
#   "cluster_name": "docker-cluster",
#   "version": { "number": "8.6.0", ... }
# }
```

浏览器访问 `http://localhost:5601` 进入 Kibana 控制台。

## 常用插件安装

### IK 中文分词器

```bash
# 进入容器安装
docker exec -it elasticsearch bash
./bin/elasticsearch-plugin install https://get.infini.cloud/elasticsearch/analysis-ik/8.6.0

# 或者将解压后的 ik 插件放到 ./plugins 目录，重启容器
docker-compose restart
```

### 安装完成后验证分词效果

```json
POST /_analyze
{
  "analyzer": "ik_max_word",
  "text": "Elasticsearch 是一个分布式搜索引擎"
}
```

## 生产环境注意事项

- 务必开启 `xpack.security.enabled=true` 并配置认证
- 根据数据量合理设置分片数和副本数
- JVM 堆内存建议不超过物理内存的 50%，且不超过 32GB
- 使用 Docker Volume 持久化数据目录 `/usr/share/elasticsearch/data`
