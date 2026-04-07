# Learning Notes - Claude Code 项目指令

## 项目概述
这是一个基于 Obsidian 的技术学习笔记仓库，涵盖云原生、算法、Go、Kafka、数据库等领域。

## 笔记规范

### 文件命名
- 使用英文小写 + 连字符作为文件名（如 `mysql-index.md`, `gmp-model.md`）
- 目录名使用英文小写 + 连字符（如 `cloud-native/`, `data-structures/`）
- 每个主题目录下的图片统一放在 `图片/` 子目录中

### 标签约定
- 每篇笔记开头使用 `#标签` 标记所属领域
- 常用标签：`#kubernetes` `#docker` `#go` `#算法` `#mysql` `#redis` `#kafka` `#elasticsearch`

### 双向链接
- 新建笔记时，必须在开头添加 `相关笔记：[[...]]` 链接到相关主题
- 更新现有笔记时，检查是否有新的关联笔记需要链接
- 使用 `[[文件名]]` 格式，不需要写完整路径

### 内容结构
- 每篇笔记应有清晰的标题层级（## 主标题，### 子标题）
- 代码示例使用 ```language 格式包裹
- 图片使用 `![[图片名.png]]` 格式引用

## 编辑约定
- 不删除有实质内容的笔记（超过 5 行有意义内容即视为有实质内容）
- 修改笔记后检查 wikilink 是否仍然有效
- 合并笔记时保留两篇笔记中不重复的内容
- README.md 作为 MOC（Map of Content），新增笔记后需同步更新

## 目录结构
```
cloud-native/  - Docker、Kubernetes 等云原生技术
  docker/      - 容器技术、镜像、网络、Dockerfile
  kubernetes/  - K8s 架构、调度、控制器
database/      - 数据库
  mysql/       - MySQL 索引、存储引擎
  redis/       - Redis 数据类型
  elasticsearch/ - ES 搜索引擎
middleware/    - 中间件
  kafka/       - Kafka 消息队列
  canal/       - MySQL Binlog 同步
  distributed-transaction/ - 分布式事务
go/            - Go 语言底层实现
algorithm/     - 算法题解和数据结构
  search/      - 搜索算法
  sorting/     - 排序算法
  techniques/  - 算法技巧
  dp/          - 动态规划
  data-structures/ - 数据结构设计
ai/            - AI/LLM 相关
misc/          - 其他工具和配置
```
