#算法 #链表

## 链表 (Linked List)

相关笔记：[[lru]] | [[lfu]]

链表是一种线性数据结构，每个节点包含数据和指向下一个节点的指针。常用技巧包括 dummy head、快慢指针、递归等。

### 链表节点定义

```go
type ListNode struct {
    Val  int
    Next *ListNode
}
```

### 常用技巧

| 技巧 | 用途 |
|:---|:---|
| Dummy Head | 简化头节点的处理，避免边界判断 |
| 快慢指针 | 找中点、检测环 |
| 递归 | 反转链表、合并链表 |

## 例题：链表相加

[2. 两数相加](https://leetcode.cn/problems/add-two-numbers/)

给你两个**非空**链表，表示两个非负整数。每位数字按**逆序**存储，每个节点只存储一位数字。将两数相加，以相同形式返回一个表示和的链表。

![[addtwonumber1.jpg]]

### 思路

使用 dummy head 简化处理，逐位相加并处理进位。

Time O(max(m, n))，Space O(max(m, n))

```go
func addTwoNumbers(l1 *ListNode, l2 *ListNode) *ListNode {
    dummy := new(ListNode) // 虚拟头节点
    head := dummy
    carry := 0

    for l1 != nil || l2 != nil || carry != 0 {
        if l1 != nil {
            carry += l1.Val
        }
        if l2 != nil {
            carry += l2.Val
        }
        head.Next = &ListNode{Val: carry % 10}
        carry /= 10
        head = head.Next
        if l1 != nil {
            l1 = l1.Next
        }
        if l2 != nil {
            l2 = l2.Next
        }
    }
    return dummy.Next
}
```
