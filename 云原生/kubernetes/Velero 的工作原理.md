**Velero 的工作原理**

**按需备份**

备份操作：

​	•	上传一个包含复制的 Kubernetes 对象的 tarball 到云对象存储。

​	•	调用云提供商的 API 来对持久卷进行磁盘快照（如果指定）。

​	•	你可以选择在备份期间执行备份钩子。例如，可能需要告诉数据库将其内存缓冲区刷新到磁盘，然后再进行快照。更多关于备份钩子的内容。

​	•	请注意，集群备份并非严格原子化。如果在备份时创建或编辑了 Kubernetes 对象，它们可能不会包含在备份中。捕获不一致信息的概率很低，但确实可能存在。

**定时备份**

定时操作允许你在定期间隔内备份数据。你可以在任何时间创建一个定时备份，第一个备份将在定时指定的间隔时间执行。这些间隔时间由 Cron 表达式指定。

Velero 保存从定时任务创建的备份，名称格式为 <SCHEDULE NAME>-<TIMESTAMP>，其中 <TIMESTAMP> 的格式为 YYYYMMDDhhmmss。有关详细信息，请参阅备份参考文档。

**备份工作流程**

当你运行 velero backup create test-backup 时：

​	•	Velero 客户端调用 Kubernetes API 服务器来创建一个 Backup 对象。

​	•	BackupController 注意到新的 Backup 对象并进行验证。

​	•	BackupController 开始备份过程。它通过查询 API 服务器来收集备份数据。

​	•	BackupController 调用对象存储服务（例如，AWS S3）上传备份文件。

默认情况下，velero backup create 会对任何持久卷进行磁盘快照。你可以通过指定附加标志来调整快照。运行 velero backup create --help 查看可用标志。可以通过选项 --snapshot-volumes=false 禁用快照。

![backup-process](/Users/hsn/Documents/backup-process.png)

**恢复**

恢复操作允许你从先前创建的备份中恢复所有对象和持久卷。你也可以只恢复对象和持久卷的过滤子集。Velero 支持多个命名空间重新映射，例如，在单次恢复中，命名空间 “abc” 中的对象可以在命名空间 “def” 下重新创建，命名空间 “123” 中的对象可以在命名空间 “456” 下重新创建。

恢复的默认名称为 <BACKUP NAME>-<TIMESTAMP>，其中 <TIMESTAMP> 的格式为 YYYYMMDDhhmmss。你也可以指定自定义名称。恢复的对象还包含一个键为 velero.io/restore-name、值为 <RESTORE NAME> 的标签。

默认情况下，备份存储位置以读写模式创建。然而，在恢复期间，你可以将备份存储位置配置为只读模式，这将禁用该存储位置的备份创建和删除。这有助于确保在恢复场景中不会意外创建或删除备份。

你可以选择在恢复期间或恢复后执行恢复钩子。例如，可能需要在数据库应用容器启动之前执行自定义的数据库恢复操作。

**恢复工作流程**

当你运行 velero restore create 时：

​	•	Velero 客户端调用 Kubernetes API 服务器创建一个 Restore 对象。

​	•	RestoreController 注意到新的 Restore 对象并进行验证。

​	•	RestoreController 从对象存储服务获取备份信息。然后，它对备份资源进行一些预处理，以确保这些资源在新集群上可以工作。例如，使用备份的 API 版本来验证恢复资源是否在目标集群上可用。

​	•	RestoreController 开始恢复过程，一次恢复每个符合条件的资源。

默认情况下，Velero 执行非破坏性恢复，这意味着它不会删除目标集群中的任何数据。如果备份中的资源在目标集群中已存在，Velero 将跳过该资源。你可以使用 --existing-resource-policy 恢复标志将 Velero 配置为使用更新策略。设置为 update 时，Velero 将尝试更新目标集群中的现有资源以匹配备份中的资源。

有关 Velero 恢复过程的更多详细信息，请参阅恢复参考页面。

**备份的 API 版本**

Velero 使用 Kubernetes API 服务器的每个组/资源的首选版本来备份资源。在恢复资源时，目标集群中必须存在相同的 API 组/版本才能使恢复成功。

例如，如果备份的集群中有一个 gizmos 资源在 things API 组中，组/版本为 things/v1alpha1、things/v1beta1 和 things/v1，并且服务器的首选组/版本为 things/v1，那么所有 gizmos 将从 things/v1 API 端点备份。当从该集群的备份恢复时，目标集群必须有 things/v1 端点才能恢复 gizmos。请注意，things/v1 不需要是目标集群中的首选版本；它只需要存在。

**设置备份过期**

当你创建备份时，可以通过添加标志 --ttl <DURATION> 来指定 TTL（生存时间）。如果 Velero 发现现有备份资源已过期，它会删除：

​	•	备份资源

​	•	云对象存储中的备份文件

​	•	所有持久卷快照

​	•	所有相关的恢复

TTL 标志允许用户使用格式为 --ttl 24h0m0s 的值来指定备份保留期（以小时、分钟和秒为单位）。如果未指定，将应用默认的 TTL 值为 30 天。

过期效果不会立即应用，而是在 gc-controller 每小时运行其对帐循环时应用。如果需要，你可以使用标志 --garbage-collection-frequency <DURATION> 调整对帐循环的频率。

如果备份未能删除，将在备份自定义资源上添加标签 velero.io/gc-failure=<Reason>。

你可以使用此标签过滤和选择未能删除的备份。

已实现的原因包括：

​	•	BSLNotFound: 备份存储位置未找到

​	•	BSLCannotGet: 备份存储位置无法从 API 服务器检索，原因不是未找到

​	•	BSLReadOnly: 备份存储位置为只读

**对象存储同步**

Velero 将对象存储视为事实来源。它不断检查以确保始终存在正确的备份资源。如果存储桶中有一个格式正确的备份文件，但 Kubernetes API 中没有相应的备份资源，Velero 会将信息从对象存储同步到 Kubernetes。

这使得在集群迁移场景中恢复功能能够工作，原始备份对象在新集群中不存在。

同样，如果 Kubernetes 中存在已完成的备份对象但在对象存储中不存在，它将从 Kubernetes 中删除，因为备份 tarball 不再存在。失败或部分失败的备份不会被对象存储同步删除。