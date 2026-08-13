# Learning Notes 协作规则

## 语言与目标

- 默认使用中文，代码、命令、API、配置键和专有名词保持原语言。
- 本仓库是个人技术学习笔记，不以求职或面试作为信息架构主轴。
- 修改前先确认知识归属、现有 MOC、相关笔记和版本边界；不要直接新建平行目录。
- 不主动暂存或提交 Git 变更，除非用户明确要求。

## 信息架构

顶层目录按领域划分：

```text
ai/            LLM 基础、推理、后训练、Agent
cloud-native/  容器与 Kubernetes
go/            Go 语言机制与 Runtime 源码
database/      MySQL、Redis、PostgreSQL、Elasticsearch
middleware/    Kafka、NATS、Canal、分布式事务
algorithm/     算法与数据结构
misc/          工具类笔记
docs/          仓库架构与维护文档
scripts/       自动校验工具
```

每个知识文件只能有一个 canonical 归属。学习路线、源码、Demo、题库属于该领域的辅助视图，不创建跨领域内容大桶。

## 文档类型

### MOC / README

必需：H1、范围说明、推荐顺序或分类、对责任范围内内容的链接。不强制标签、Mermaid 和面试问答。

### 主题知识笔记

建议结构：

```markdown
#tag

相关笔记：[[related-note]]

# 标题

## 概述
## 核心机制
## 实践或示例
## 自检问题
## 参考与版本边界
```

机制存在多节点关系时使用 Mermaid；没有重要关系时不为了形式强行画图。`## 面试要点` 是可选复习卡片，不再是所有文档的硬性要求。

### 源码导读

在主题笔记要求之外，必须注明项目版本、commit 或审查日期、源码入口、关键调用链和验证方式。不要把易漂移的 `master` 行号写成永久事实。

### 学习路线 / 进度

必须说明目标人群、前置条件、阶段产出和完成标准。不强制面试问答。

### Demo 手册

必须说明环境、运行命令、预期结果、最后验证环境/日期和已知限制。代码与说明保持同目录，整体移动。

### 题库

使用“问题 → 考察点 → 回答骨架 → 设计延伸 → 常见误区”，并链接 canonical 笔记，不复制技术正文。

## 链接与资源

- 普通 Markdown 文件名使用全局唯一的英文小写 kebab-case；`README.md` 例外。
- 新增、删除、重命名或移动后运行 `python3 scripts/validate_notes.py`。
- Wikilink 使用 `[[file-name]]`、`[[file-name|显示名]]` 或 `[[file-name#标题]]`，不要写 `\|`。
- 图片保存在所属主题附近的 `图片/` 目录；图片 basename 必须全局唯一。
- 兼容 GitHub 的页面优先使用相对 Markdown 图片链接；使用 Obsidian embed 时必须保证校验器可唯一解析。
- 根 README 只维护一级入口；新增笔记更新最近的所属 MOC，不要求根 README 罗列每篇内容。

## 修改流程

1. 确定文档类型与唯一归属。
2. 查找已有笔记，明确概念页、组件页、源码页和 Demo 的职责边界。
3. 更新正文及最近的 MOC。
4. 运行结构校验；必要时运行 `python3 scripts/validate_notes.py --quality` 查看历史质量债务。
5. 检查 `git diff --check` 与 `git status --short`，确认未自动暂存。

## 禁止事项

- 不创建空重定向笔记或只有链接的 stub；移动时尽量保留 basename。
- 不为了“统一”机械合并不同深度的概念页、组件页和源码页。
- 不手工维护容易漂移的全库精确篇数。
- 不把“命令成功退出”“已编译”写成完整实验成功；应记录预期输出、运行状态与限制。

完整架构说明见 `docs/architecture.md`。
