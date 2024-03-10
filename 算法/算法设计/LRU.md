#LRU #缓存策略
## Least Recently Used (LRU) Cache

LRU缓存是一种常见的缓存替换策略，它基于最近访问的时间来淘汰最近最少使用的缓存条目。在LRU缓存中，每次访问一个缓存条目时，它被移动到最近使用的位置（通常是在链表的头部），并且当缓存容量达到限制时，最少使用的条目将被淘汰。
## 实现
在代码中，LRU缓存使用了双向链表（`list.List`）来维护缓存条目的顺序，以及一个映射（`keyToNode`）来快速查找缓存中的条目。具体实现了 `Get` 方法用于获取缓存条目，以及 `Put` 方法用于插入或更新缓存条目。
```go
type entry struct {
    key, value int
}
type LRUCache struct {
    capacity  int
    list      *list.List // 双向链表
    keyToNode map[int]*list.Element
}
func Constructor(capacity int) LRUCache {
    return LRUCache{capacity, list.New(), map[int]*list.Element{}}
}
func (c *LRUCache) Get(key int) int {
    node := c.keyToNode[key]
    if node == nil { // 没有这本书
        return -1
    }
    c.list.MoveToFront(node) // 把这本书放在最上面
    return node.Value.(entry).value
}

func (c *LRUCache) Put(key, value int) {
    if node := c.keyToNode[key]; node != nil { // 有这本书
        node.Value = entry{key, value} // 更新
        c.list.MoveToFront(node) // 把这本书放在最上面
        return
    }
    c.keyToNode[key] = c.list.PushFront(entry{key, value}) // 新书，放在最上面
    if len(c.keyToNode) > c.capacity { // 书太多了
        delete(c.keyToNode, c.list.Remove(c.list.Back()).(entry).key) // 去掉最后一本书
    }
}
```
## 方法解析

- `Constructor(capacity int) LRUCache`：LRUCache 的构造函数，初始化了 LRUCache 结构体及其成员变量。

- `(c *LRUCache) Get(key int) int`：获取指定 key 对应的 value 值，并将该条目移动到链表头部，以表示最近使用。

- `(c *LRUCache) Put(key, value int)`：将一个新的 key-value 对插入到缓存中，若缓存已满，则淘汰最近最少使用的条目
## 优点

- 时间复杂度较低：LRU缓存的查询和更新操作的时间复杂度都为 O(1)，因为通过映射和链表的结构可以快速定位和更新缓存条目。

- 缓存淘汰策略合理：LRU算法根据最近的访问情况进行淘汰，相对公平地释放缓存空间。