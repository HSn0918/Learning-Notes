#深度优先
## 深度优先搜索
  
深度优先搜索（DFS，Depth-First Search）是一种用于遍历或搜索树或图的算法。这个名称源于它尝试尽可能深地向前探索，直到到达终点，然后通过回溯来探索其他分支。DFS可以使用递归或栈来实现，在不同的应用场景中有着广泛的应用，比如解决迷宫问题、路径寻找、检测循环等。
  
### DFS 的工作原理

1. **选择一个起点**：在树或图中选择一个起点作为搜索的开始。
2. **向深处探索**：从起点开始，沿着某一路径尽可能深地探索，直到到达一个节点，该节点没有未访问的相邻节点，或达到特定条件（如到达终点）。
3. **回溯**：当到达一个没有未访问相邻节点的节点时，算法回溯到上一个节点，寻找是否有其他的路径可以继续探索。
4. **重复步骤 2 和 3**：重复这个过程，直到所有的节点都被访问过，或找到所需的路径或解决方案。

### 递归实现

DFS 最简单的实现方式是使用递归。在递归的每一步，你都选择一个未被访问的邻接节点，向它前进，并继续这个过程。当没有更多的邻接节点可以访问时，递归开始回溯，寻找其他可能的路径。

### 栈实现

DFS 也可以通过使用栈来非递归地实现。算法的核心思想不变，只是使用了栈来显式地跟踪访问路径，而不是依赖于递归调用的系统栈。

1. **将起点压入栈中**。
2. **循环直到栈为空**：在每次循环中，从栈中弹出一个节点作为当前节点。
3. **访问当前节点**：如果当前节点是目标节点，或需要进行特定处理，则进行相应的操作。
4. **将相邻的未访问节点压入栈中**。
## 模版
```go
func isValidBST(root *TreeNode) bool {  
    pre := math.MinInt  
    var dfs func(*TreeNode) bool  
    //中序遍历 左中右
    dfs = func(node *TreeNode) bool {  
       if node == nil {  
          return true  
       }  
       if !dfs(node.Left) || node.Val <= pre {  
          return false  
       }  
       pre = node.Val  
       return dfs(node.Right)  
    }  
    return dfs(root)  
}
```

笛卡尔积
```go
package main

import (
	"fmt"
)

func main() {
	arrays := [][]string{
		{"a", "b"},
		{"1", "2"},
		{"x", "y"},
	}
	result := cartesianProduct(arrays)
	for _, combo := range result {
		fmt.Println(combo)
	}
}

// cartesianProduct 计算多个字符串数组的笛卡尔积
func cartesianProduct(arrays [][]string) [][]string {
	if len(arrays) == 0 {
		return [][]string{{}}
	}
	
	// 递归地获取后面的笛卡尔积
	rest := cartesianProduct(arrays[1:])
	
	var result [][]string
	for _, val := range arrays[0] {
		for _, r := range rest {
			result = append(result, append([]string{val}, r...))
		}
	}
	return result
}
```