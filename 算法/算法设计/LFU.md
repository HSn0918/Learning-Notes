#LFU #缓存策略
## Least Frequently Used, LFUCache
缓存淘汰策略的数据结构。LFU 缓存淘汰策略会根据数据被访问的频率来决定淘汰哪些数据，优先淘汰那些被访问次数最少的数据。与最近最少使用（Least Recently Used, LRU）缓存淘汰策略不同，LFU 关注的是访问频率而不是访问的时间顺序。
## 实现
### 结构体定义

- `LFUCache` 结构体定义了几个关键的属性：
    - `capacity`：缓存的容量。
    - `minFreq`：当前缓存中所有条目最小的访问频率。
    - `keyToNode`：一个映射，将键映射到双向链表中的节点，用于快速访问和修改键对应的条目。
    - `freqToList`：一个映射，将访问频率映射到一个双向链表，链表中存储的是具有相同访问频率的条目。
### 方法实现

- `Constructor(capacity int) LFUCache`：构造函数，初始化一个 LFU 缓存实例。
- `pushFront(e *entry)`：辅助方法，将一个条目按照其频率添加到对应频率的双向链表的前端。
- `getEntry(key int) *entry`：辅助方法，根据键获取条目。如果条目存在，则将其频率增加，并移动到对应频率的双向链表的前端。如果因为频率的增加而导致原来的频率链表为空，则需要做相应的处理，如删除空链表，调整最小频率等。
- `Get(key int) int`：根据键获取值。如果键存在，则返回相应的值，并更新访问频率；如果键不存在，则返回 -1。
- `Put(key, value int)`：添加或更新一个键值对。如果键已存在，则更新其值和访问频率；如果键不存在，则需要添加新的条目。如果添加新条目导致缓存容量超限，需要淘汰最不经常使用的条目，即访问频率最低的条目中最旧的一个。
### 特别处理

- 初始化时，通过 `init()` 方法设置 `debug.SetGCPercent(-1)` 禁用了垃圾回收，这通常是为了提高性能，避免在高频操作缓存时触发频繁的垃圾回收。
### 工作流程
1. 当请求获取某个键的值时，`Get` 方法会查看该键是否存在：
    - 如果存在，更新该键的访问频率，并返回其值。
    - 如果不存在，返回 -1。
2. 当需要放入一个键值对时，`Put` 方法会判断该键是否已存在：
    - 如果存在，更新该键的值和访问频率。
    - 如果不存在，首先检查是否达到容量上限：
        - 如果达到上限，先淘汰一个最不经常使用的条目，然后添加新条目。
        - 如果未达上限，直接添加新条目。
3. 在添加或更新条目的访问频率时，条目会被移动到相应频率的链表前端。如果某个频率的链表变为空，则进行相应的清理操作，并可能调整 `minFreq` 的值。
```go

type entry struct {
    key, value, freq int // freq 表示这本书被看的次数
}

type LFUCache struct {
    capacity   int
    minFreq    int
    keyToNode  map[int]*list.Element
    freqToList map[int]*list.List
}

func Constructor(capacity int) LFUCache {
    return LFUCache{
        capacity:   capacity,
        keyToNode:  map[int]*list.Element{},
        freqToList: map[int]*list.List{},
    }
}

func (c *LFUCache) pushFront(e *entry) {
    if _, ok := c.freqToList[e.freq]; !ok {
        c.freqToList[e.freq] = list.New() // 双向链表
    }
    c.keyToNode[e.key] = c.freqToList[e.freq].PushFront(e)
}

func (c *LFUCache) getEntry(key int) *entry {
    node := c.keyToNode[key]
    if node == nil { // 没有这本书
        return nil
    }
    e := node.Value.(*entry)
    lst := c.freqToList[e.freq]
    lst.Remove(node) // 把这本书抽出来
    if lst.Len() == 0 { // 抽出来后，这摞书是空的
        delete(c.freqToList, e.freq) // 移除空链表
        if c.minFreq == e.freq {
            c.minFreq++
        }
    }
    e.freq++
    c.pushFront(e) // 放在右边这摞书的最上面
    return e
}

func (c *LFUCache) Get(key int) int {
    if e := c.getEntry(key); e != nil { // 有这本书
        return e.value
    }
    return -1
}

func (c *LFUCache) Put(key, value int) {
    if e := c.getEntry(key); e != nil { // 有这本书
        e.value = value // 更新 value
        return
    }
    if len(c.keyToNode) == c.capacity { // 书太多了
        lst := c.freqToList[c.minFreq] // 最左边那摞书
        delete(c.keyToNode, lst.Remove(lst.Back()).(*entry).key) // 移除这摞书的最下面的书
        if lst.Len() == 0 { // 这摞书是空的
            delete(c.freqToList, c.minFreq) // 移除空链表
        }
    }
    c.pushFront(&entry{key, value, 1}) // 新书放在「看过 1 次」的最上面
    c.minFreq = 1
}

func init() { debug.SetGCPercent(-1) }

```
