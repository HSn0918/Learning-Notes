#基数排序

## 基数排序 (Radix Sort)

相关笔记：[[sorting-overview]] | [[merge-sort]]

基数排序（Radix Sort）是一种非比较型整数排序算法，原理是将整数按位数切割成不同的数字，然后按每个位数分别排序。由于整数也可以表示字符串或浮点数，所以基数排序并不仅限于整数。

### 算法流程

```mermaid
graph LR
    A["原始数组"] --> B["按个位排序"]
    B --> C["按十位排序"]
    C --> D["按百位排序"]
    D --> E["... 直到最高位"]
    E --> F["排序完成"]
```

### 具体步骤

1. **找到最大数，确定最大位数**：遍历数组找到最大的数，最大数的位数决定了排序的轮数
2. **按位排序**：从最低位开始，使用 counting sort 对当前位进行稳定排序
3. **收集**：将排序后的内容收集起来
4. **重复步骤 2 和 3**：直到最高位排序完成

### 复杂度分析

| 指标 | 复杂度 |
|:---:|:---:|
| Time | O(N * K)，K 为最大位数 |
| Space | O(N + B)，B 为基数（十进制为 10） |
| 稳定性 | 稳定 |

### 优缺点

**优点**：
- 稳定的排序算法
- 在位数不大时速度非常快

**缺点**：
- 需要额外的存储空间
- 主要适用于整数或可以转换为整数的类型

## 模版

```go
const BASE int = 10
const MAXN int = 50001

var help [MAXN]int
var cnts [BASE]int

func sortArray(arr []int) []int {
    n := len(arr)
    if n > 1 {
        // 找到数组中的最小值，将所有数转为非负数
        min := arr[0]
        for i := 1; i < n; i++ {
            if arr[i] < min {
                min = arr[i]
            }
        }
        max := 0
        for i := 0; i < n; i++ {
            arr[i] -= min
            if arr[i] > max {
                max = arr[i]
            }
        }
        // 基数排序
        radixSort(arr, n, bits(max))
        // 还原
        for i := 0; i < n; i++ {
            arr[i] += min
        }
    }
    return arr
}

func bits(number int) int {
    ans := 0
    for number > 0 {
        ans++
        number /= BASE
    }
    return ans
}

func radixSort(arr []int, n int, bits int) {
    for offset, bitCnt := 1, bits; bitCnt > 0; offset *= BASE {
        for i := range cnts {
            cnts[i] = 0
        }
        for i := 0; i < n; i++ {
            cnts[(arr[i]/offset)%BASE]++
        }
        // 前缀和，将 count 转为位置索引
        for i := 1; i < BASE; i++ {
            cnts[i] += cnts[i-1]
        }
        // 从后往前遍历保证稳定性
        for i := n - 1; i >= 0; i-- {
            cnts[(arr[i]/offset)%BASE]--
            help[cnts[(arr[i]/offset)%BASE]] = arr[i]
        }
        for i := 0; i < n; i++ {
            arr[i] = help[i]
        }
        bitCnt--
    }
}
```
