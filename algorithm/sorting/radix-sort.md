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

## 面试要点

### 高频问题

**Q: 基数排序为什么能突破 O(N log N) 下界？**

> [!question]- 参考答案（点击展开）
>
> O(N log N) 是基于「两两元素比较」的排序的理论下界，而 radix sort 属于非比较型排序，它不通过比较元素大小来确定顺序，而是按位（数字、字符）借助 counting sort 桶分配。其时间是 O(N * K)，K 为最大位数（轮数），因此当 K 远小于 log N 时可以做到线性级别，并不违反比较排序下界。

**Q: 基数排序应该从最低位（LSD）还是最高位（MSD）开始？两者有什么区别？**

> [!question]- 参考答案（点击展开）
>
> 笔记中模版采用 LSD（Least Significant Digit），`offset` 从 1（个位）开始、每轮 `*= BASE` 逐位向高位推进，每一轮都对全体数据做一次稳定排序，实现简单、逻辑统一。MSD 从最高位开始，按位递归分桶，每个桶内部再独立排序，更适合变长字符串和提前终止（前缀确定后即可剪枝），但实现复杂、需要递归。定长整数排序通常用 LSD。

**Q: 为什么基数排序里每一轮的子排序必须是稳定的？**

> [!question]- 参考答案（点击展开）
>
> LSD 是先按低位排好序，再按高位排序。如果高位相同，必须保留上一轮（低位）已排好的相对顺序，否则低位的排序结果会被破坏，整体正确性不成立。因此每一轮必须用 stable sort（这里是 counting sort）。模版中「前缀和定位 + 从后往前遍历」正是为了保证 counting sort 的稳定性。

**Q: 基数排序的时间和空间复杂度是多少？稳定吗？**

> [!question]- 参考答案（点击展开）
>
> 时间 O(N * K)，K 是最大数的位数（轮数），每轮做一次 O(N + B) 的 counting sort。空间 O(N + B)，对应辅助数组 help（O(N)）与计数桶 cnts（O(B)，B 为基数，十进制为 10）。基数排序是稳定排序。

**Q: 笔记的模版里为什么要先减去 min、最后再加回 min？**

> [!question]- 参考答案（点击展开）
>
> 模版用 `(arr[i]/offset)%BASE` 取每一位，要求元素非负，负数取模会得到错误的桶下标。因此先遍历找到最小值 min，把所有数平移成非负数（`arr[i] -= min`），排序完成后再统一加回（`arr[i] += min`）。平移是单调变换，不改变元素间相对大小，从而在不破坏正确性的前提下兼容负数输入。

**Q: 这里的 BASE（基数）取值会如何影响性能？BASE 越大越好吗？**

> [!question]- 参考答案（点击展开）
>
> BASE 越大，单个数的位数 K（轮数）越少，但每轮 counting sort 的桶空间 O(B) 与清零开销越大。整体大致是 O((N + B) * log_B(max))，存在权衡：BASE 太小则轮数多，太大则桶空间和清零开销大。工程上常取 256（按字节处理），且当 BASE = 2^k 时可用位移和掩码替代除法/取模，进一步提速。

**Q: counting sort 中「前缀和」那一步起什么作用？**

> [!question]- 参考答案（点击展开）
>
> 先统计每个桶（每个数字 0~9）出现的次数，再对 cnts 做前缀和，使 cnts[i] 表示「当前位数字 ≤ i 的元素总数」，即数字 i 这一段在输出数组中的右边界（exclusive，恰好是末尾下标 +1）。随后从后往前遍历，对每个元素先把对应 cnts 减一再写入该下标，把计数直接转换成稳定的目标位置——倒序遍历配合「先减后放」正是稳定性的来源。

### 面试加分点

- 能区分 LSD 与 MSD，并说明 MSD 适合变长字符串、可按前缀提前剪枝；LSD 实现统一、对定长整数更高效。
- 指出基数排序对负数的处理思路：平移法（减 min）或对最高位做符号位特殊处理，避免取模出错。
- 理解 BASE 选择的工程权衡，并能说明取 BASE = 2^k（如 256）时可用位移和掩码替代除法/取模，大幅提速。
- 能讲清「基数排序 = 多轮稳定的 counting sort」，并强调稳定性是 LSD 正确性的前提，而非可选优化。
- 知道基数排序的局限：依赖键可拆分为定长的位/字符，浮点数需特殊编码（如对 IEEE 754 翻转符号位/对负数翻转全部位），不适合任意可比较类型，且有额外空间开销。
- 模版用了全局静态数组 help/cnts（MAXN 固定上限）以避免重复分配，能讨论这种写法在 LeetCode 场景下的性能优势，以及它对输入规模上限（n ≤ MAXN）的隐含假设与潜在越界风险。
