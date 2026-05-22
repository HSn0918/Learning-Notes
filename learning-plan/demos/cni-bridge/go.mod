module learning-plan/demos/cni-bridge

go 1.21

// 本 demo 是一个**教学骨架**，故意不引入任何外部依赖——只用 encoding/json + os + fmt + io。
//
// 真正的 CNI 插件（参考 containernetworking/plugins）会用到：
//   - github.com/vishvananda/netlink        // 创建 veth、操作 link 状态
//   - github.com/vishvananda/netns          // setns(2) 切换 namespace
//   - github.com/containernetworking/cni    // libcni 的 skel/types/current 等公共抽象
//   - github.com/coreos/go-iptables/iptables // iptables 规则操作（如 ipMasq）
//
// 如果你想把这个 demo 改成"在 Linux 节点上真的能跑"，依赖列表大致是：
//
//   require (
//       github.com/containernetworking/cni v1.1.2
//       github.com/vishvananda/netlink v1.2.1
//   )
//
// 然后 main.go 里的 cmdAdd 把 dummy IP 的部分换成 netlink 真实操作即可——主流程
// （读 env、读 stdin、按 CNI_COMMAND 分发、输出 Result JSON）完全不变。
