# 算法 (Algorithm) · 学习索引

算法学习的核心不是背模板，而是建立“数据结构—状态—复杂度—边界条件”的解题模型。本索引覆盖搜索图论、排序、常用技巧、动态规划、核心数据结构与字符串匹配六个方向，面试题只作为检验理解的一种方式。

> ⬆ 返回 [知识库首页](../README.md)

## 🧭 推荐学习顺序

**入门（搭骨架，先建立遍历与排序的基本直觉）**
Binary Search 二分查找 → BFS 广度优先搜索 → DFS 深度优先搜索 → 链表 Linked List → Sorting 排序算法总结 → Quick Sort 快速排序 → Merge Sort 归并排序

**进阶（高频考点，从模板套用到独立建模）**
Backtracking 回溯算法 → 前缀和 Prefix Sum → 差分数组 Difference Array → 滑动窗口 Sliding Window → 单调栈 Monotonic Stack → House Robber 打家劫舍（DP 入门）→ Heap Sort 堆排序 → XOR 异或运算 → Bitwise AND 位运算技巧 → LRU Cache

**深入（区分度高，理解门槛与建模难度较大）**
图论 BFS/DFS 应用 → Topological Sort 拓扑排序 → Dijkstra 最短路径 → Union-Find 并查集 → LCA 最低公共祖先 → Unbounded Knapsack 完全背包 → Radix Sort 基数排序 → LFU Cache → Trie 前缀树 → KMP 字符串匹配

## 📚 笔记清单

### 搜索与图论

| 笔记 | 简介 | 难度 |
|------|------|------|
| [BFS 广度优先搜索](search/bfs.md) | 用 queue 逐层遍历图/树，FIFO 扩展邻居，适合求无权图最短路径与层序遍历 | 入门 |
| [DFS 深度优先搜索](search/dfs.md) | 用递归或 stack 尽可能深入再回溯，遍历图/树的所有路径，是回溯与连通性问题的基础 | 入门 |
| [Backtracking 回溯算法](search/backtracking.md) | 基于 DFS 穷举所有选择，做选择-递归-撤销选择，解决排列组合、子集等搜索问题 | 进阶 |
| [Binary Search 二分查找](search/binary-search.md) | 在有序数组中 O(log n) 定位目标，处理第一次/最后一次出现等边界变体 | 入门 |
| [LCA 最低公共祖先](search/lca.md) | 在二叉树/BST 中后序递归查找两节点最近共同祖先，含多种树结构变种 | 进阶 |
| [图论基础与 BFS/DFS 应用](search/graph-bfs-dfs.md) | 讲解邻接矩阵/邻接表表示及 BFS/DFS 在图遍历、连通性问题上的落地 | 进阶 |
| [Dijkstra 最短路径](search/dijkstra.md) | 基于贪心+松弛的单源最短路径算法，适用于非负边权加权图 | 深入 |
| [Topological Sort 拓扑排序](search/topological-sort.md) | 对 DAG 做线性排序，Kahn 入度法与 DFS 逆后序两种实现，可检测环 | 深入 |
| [Union-Find 并查集](search/union-find.md) | 管理元素分组的数据结构，路径压缩+按秩合并把 Union/Find 降到近 O(1) | 深入 |

### 排序

| 笔记 | 简介 | 难度 |
|------|------|------|
| [排序算法总结 Sorting Overview](sorting/sorting-overview.md) | 对比八大排序的时间/空间复杂度与稳定性，建立排序选型全局视图 | 入门 |
| [Quick Sort 快速排序](sorting/quick-sort.md) | 分治+随机 pivot 分区，结合荷兰国旗三路分区处理重复元素，平均 O(N logN) | 进阶 |
| [Merge Sort 归并排序](sorting/merge-sort.md) | 分治归并的稳定 O(N logN) 排序，附 Master 公式推导递归复杂度 | 进阶 |
| [Heap Sort 堆排序](sorting/heap-sort.md) | 基于完全二叉树的大/小根堆，数组实现的原地 O(N logN) 排序 | 进阶 |
| [Radix Sort 基数排序](sorting/radix-sort.md) | 非比较型整数排序，按位用计数排序稳定分配，达到线性时间复杂度 | 深入 |

### 技巧

| 笔记 | 简介 | 难度 |
|------|------|------|
| [前缀和 Prefix Sum](techniques/prefix-sum.md) | 预处理累加和实现 O(1) 区间求和，常配合 HashMap 解子数组和问题 | 进阶 |
| [差分数组 Difference Array](techniques/diff-array.md) | 前缀和的逆运算，把区间批量加操作优化为 O(1)，适合频繁区间修改 | 进阶 |
| [滑动窗口 Sliding Window](techniques/sliding-window.md) | 双指针维护动态区间，解决子串/子数组最优化问题，模板化收缩窗口 | 进阶 |
| [单调栈 Monotonic Stack](techniques/monotonic-stack.md) | 维护单调递增/减的栈，O(N) 解决 Next Greater Element 类问题 | 进阶 |
| [XOR 异或运算](techniques/xor.md) | 无进位相加性质，靠成对抵消找出唯一/奇数次出现的数，区间异或和技巧 | 进阶 |
| [位运算 AND 技巧 Bitwise AND](techniques/bitwise-and.md) | 利用 n&(n-1) 消除最低位 1，Brian Kernighan 算法高效统计 1 的个数 | 进阶 |

### 动态规划

| 笔记 | 简介 | 难度 |
|------|------|------|
| [打家劫舍 House Robber](dp/house-robber.md) | 经典线性 DP，不能偷相邻房屋，f(i)=max(f(i-1), f(i-2)+nums[i]) 状态转移 | 进阶 |
| [完全背包 Unbounded Knapsack](dp/unbounded-knapsack.md) | 物品可无限选取，靠内层容量从小到大遍历实现重复选取的背包 DP | 深入 |

### 数据结构

| 笔记 | 简介 | 难度 |
|------|------|------|
| [LRU Cache 最近最少使用缓存](data-structures/lru.md) | HashMap + 双向链表实现 O(1) 读写，淘汰最久未访问条目，高频手撕题 | 进阶 |
| [LFU Cache 最不经常使用缓存](data-structures/lfu.md) | 按访问频率淘汰，频率桶+双向链表+minFreq 维持 O(1) 读写，难度高于 LRU | 深入 |
| [链表 Linked List](data-structures/linked-list.md) | 线性指针结构，dummy head、快慢指针、迭代反转等高频技巧汇总 | 入门 |

### 字符串

| 笔记 | 简介 | 难度 |
|------|------|------|
| [KMP 字符串匹配](string/kmp.md) | 靠 next 前缀函数避免文本指针回退，把字符串匹配优化到 O(n+m) | 深入 |
| [Trie 前缀树 Prefix Tree](string/trie.md) | 树形结构存储字符串集合，前缀查询 O(m) 与集合大小无关，适合自动补全 | 深入 |

---
共 **27** 篇 · 入门 5 / 进阶 14 / 深入 8
