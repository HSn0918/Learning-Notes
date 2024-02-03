#unionfs
# Union FS
## 文件系统
- 将不同目录挂载到同一个虚拟文件系统下 (unite several directories into a single virtual filesystem)的文件系统
- 支持为每一个成员目录(类似Git Branch )设定 readonly、readwrite 和 whiteout-able 权限
- 文件系统分层对readonly 权限的 branch 可以逻辑上进行修改增量地,不影响 readonly 部分的)
- 通常 Union FS 有两个用途一方面可以将多个 disk 挂到同一个目录下,另一个更常用的就是将一个readonly的 branch和一个 writeable的 branch 联合在一起
