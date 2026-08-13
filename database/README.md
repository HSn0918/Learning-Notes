# 数据库 (Database) · 学习索引

数据库学习从“数据如何存、查、并发修改和恢复”四个问题展开。MySQL 用来理解索引、事务与锁，Redis 补充缓存和高可用，Elasticsearch 与 PostgreSQL 分别扩展检索系统和关系数据库高级能力。

> ⬆ 返回 [知识库首页](../README.md)

## 🧭 推荐学习顺序

**入门（建立全局认知）**
MySQL 存储引擎对比 (InnoDB vs MyISAM) → Redis 五大数据类型与底层结构 → Elasticsearch 核心概念与架构 → PostgreSQL 多进程架构与核心原理

**进阶（掌握核心机制）**
MySQL 索引原理 (B+Tree/聚簇索引) → MySQL 事务与 MVCC → Redis 持久化 (RDB/AOF) → Redis 缓存三大问题 (穿透/击穿/雪崩) → ES 字段类型与映射 → ES + Kibana Docker 部署

**深入（攻克难点与高可用）**
MySQL 锁机制 (Next-Key Lock/间隙锁) → Redis 高可用 (主从/Sentinel/Cluster) → PostgreSQL 高级特性 (窗口函数/锁/复制)

## 📚 笔记清单

### MySQL

| 笔记 | 简介 | 难度 |
|------|------|------|
| [MySQL 存储引擎 (InnoDB vs MyISAM)](mysql/mysql-engine.md) | 对比 InnoDB、MyISAM、Memory 三大引擎在事务、锁粒度、索引、崩溃恢复上的差异 | 入门 |
| [MySQL 索引 (B+Tree/聚簇索引)](mysql/mysql-index.md) | 从数据结构、存储、字段维度分类讲解索引，重点剖析 InnoDB 聚簇索引与 B+Tree 结构 | 进阶 |
| [MySQL 事务与 MVCC](mysql/mysql-transaction.md) | 讲解 ACID、四种隔离级别及 MVCC 通过隐藏列与 Read View 实现并发控制的原理 | 进阶 |
| [MySQL 锁机制 (Next-Key Lock)](mysql/mysql-lock.md) | 梳理全局锁、表锁、行锁层级，深入 Record/Gap/Next-Key Lock 如何解决幻读 | 深入 |

### Redis

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Redis 数据类型与底层结构](redis/redis-data-types.md) | 讲解 String/List/Hash/Set/ZSet 五大类型及 SDS/quicklist/skiplist 等底层实现与场景 | 入门 |
| [Redis 持久化 (RDB/AOF)](redis/redis-persistence.md) | 对比 RDB 快照与 AOF 日志，剖析 BGSAVE 的 fork+COW 流程与混合持久化机制 | 进阶 |
| [Redis 缓存三大问题](redis/redis-cache.md) | 讲解缓存穿透、击穿、雪崩的成因与布隆过滤器、互斥锁、过期打散等解决方案 | 进阶 |
| [Redis 高可用 (主从/Sentinel/Cluster)](redis/redis-cluster.md) | 对比主从复制、Sentinel 哨兵、Redis Cluster 三种高可用方案及全量/增量同步流程 | 深入 |

### Elasticsearch

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Elasticsearch 基础与架构](elasticsearch/elasticsearch-basics.md) | 介绍基于 Lucene 的分布式搜索引擎及 Cluster/Node/Index/Shard/Replica 核心概念 | 入门 |
| [Elasticsearch 字段类型与映射](elasticsearch/es-field-types.md) | 讲解 text/keyword、数值、日期、object/nested 等字段类型对查询与存储的影响 | 进阶 |
| [ES + Kibana Docker 部署](elasticsearch/es-kibana-docker.md) | 用 Docker Compose 搭建 Elasticsearch 8.6 + Kibana 开发环境的实操步骤 | 入门 |

### PostgreSQL

| 笔记 | 简介 | 难度 |
|------|------|------|
| [PostgreSQL 基础与多进程架构](postgresql/postgresql-basics.md) | 讲解 PG 多进程模型、Postmaster/Backend 进程、后台 Worker 与共享内存核心原理 | 入门 |
| [PostgreSQL 高级特性 (窗口函数/复制)](postgresql/postgresql-advanced.md) | 讲解窗口函数、锁机制、流复制、连接池等高级特性，对标 MySQL 的进阶能力 | 深入 |

---
共 **13** 篇 · 入门 5 / 进阶 5 / 深入 3
