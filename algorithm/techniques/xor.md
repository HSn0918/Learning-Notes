#位运算

## 异或运算 (XOR)

相关笔记：[[bitwise-and]]

### 异或运算性质

1. 异或运算就是**无进位相加**
2. 满足交换律、结合律 -- 同一批数字不管异或顺序是什么，最终结果都一样
3. `0 ^ n = n`，`n ^ n = 0`
4. 整体异或和为 `x`，某部分异或和为 `y`，则剩下部分异或和为 `x ^ y`

其中第 1 条最重要，所有其他结论都可以由此推导。第 4 条相关的题目最多，利用区间上异或和的性质。

### Brian Kernighan 算法

提取二进制状态中最右侧的 1：`rightOne = n & (-n)`

## 题目

### 题目 1：交换两个数

不使用额外变量交换两个数：

```go
func swap(a, b *int) {
    *a = *a ^ *b
    *b = *a ^ *b
    *a = *a ^ *b
}
```

### 题目 2：不用判断语句返回最大值

利用符号位判断大小：

```go
func max(a, b int) int {
    c := a - b
    k := (c >> 31) & 1
    return a - k*c
}
```

### 题目 3：找到缺失的数字

数组 `[0, n]` 中缺失一个数，利用索引和值异或抵消：

```go
func missingNumber(nums []int) int {
    xor := 0
    for i := 0; i < len(nums); i++ {
        xor ^= i ^ nums[i]
    }
    return xor ^ len(nums)
}
```

### 题目 4：出现奇数次的数（1 种）

数组中 1 种数出现奇数次，其他数都出现偶数次。全部异或即可：

```go
func findOdd(nums []int) int {
    xor := 0
    for _, num := range nums {
        xor ^= num
    }
    return xor
}
```

### 题目 5：出现奇数次的数（2 种）

数组中有 2 种数出现奇数次。先全部异或得到两数的异或值，再用最低位的 1 将数组分成两组：

```go
func findTwoOdds(nums []int) (int, int) {
    xor := 0
    for _, num := range nums {
        xor ^= num
    }
    rightOne := xor & (-xor) // 获取最右侧的1
    onlyOne := 0
    for _, num := range nums {
        if (num & rightOne) != 0 {
            onlyOne ^= num
        }
    }
    return onlyOne, xor ^ onlyOne
}
```

### 题目 6：出现次数少于 m 次的数

数组中只有 1 种数出现次数少于 m 次，其他数都出现了 m 次。统计每一位上 1 的个数，对 m 取模即可：

```go
func findLessThanMTimes(nums []int, m int) int {
    bits := make([]int, 32)
    for _, num := range nums {
        for j := 0; j < 32; j++ {
            bits[j] += (num >> j) & 1
        }
    }
    result := 0
    for i, count := range bits {
        if count%m != 0 {
            result |= 1 << i
        }
    }
    return result
}
```
