// cni-bridge demo —— 一个最小可运行的 CNI 插件骨架。
//
// 真正的 bridge plugin（containernetworking/plugins/main/bridge）会调用 netlink
// 创建 veth pair、操作 network namespace、调 IPAM 子插件分配 IP。本 demo 把这些
// 平台相关的系统调用全部去掉，**只演示 CNI 协议骨架本身**：
//
//   1. 读 stdin JSON 配置
//   2. 读 CNI_COMMAND / CNI_CONTAINERID / CNI_NETNS / CNI_IFNAME / CNI_ARGS 环境变量
//   3. 按操作类型分发：
//        ADD     -> 输出一份"假装分配成功"的 Result JSON（含 dummy IP）
//        DEL     -> 直接成功退出（幂等）
//        CHECK   -> 直接成功退出
//        VERSION -> 输出 supportedVersions
//        GC      -> 直接成功退出（v1.1 新增）
//   4. 出错时按 spec 输出 ErrorReply JSON 到 stdout，进程以非零 exit code 退出
//
// 测试方法（无须 root，因为没有真的操作 netns）：
//
//	cat > /tmp/cfg.json <<'EOF'
//	{
//	  "cniVersion": "1.0.0",
//	  "name": "demo-net",
//	  "type": "cni-bridge",
//	  "bridge": "demo0",
//	  "ipam": {"type": "host-local", "subnet": "10.244.0.0/24"}
//	}
//	EOF
//
//	CNI_COMMAND=VERSION ./cni-bridge
//
//	CNI_COMMAND=ADD CNI_CONTAINERID=test-001 CNI_NETNS=/tmp/fake-ns \
//	  CNI_IFNAME=eth0 CNI_PATH=/opt/cni/bin \
//	  CNI_ARGS="K8S_POD_NAMESPACE=default;K8S_POD_NAME=nginx" \
//	  ./cni-bridge < /tmp/cfg.json
//
//	CNI_COMMAND=DEL CNI_CONTAINERID=test-001 CNI_NETNS=/tmp/fake-ns \
//	  CNI_IFNAME=eth0 ./cni-bridge < /tmp/cfg.json
//
// K8s 实际调用路径：
//
//   kubelet --(CRI RunPodSandbox)--> containerd
//   containerd 读 /etc/cni/net.d/10-demo.conflist，按链顺序对每个 plugin：
//     fork+exec /opt/cni/bin/cni-bridge
//                  env: CNI_COMMAND=ADD, CNI_CONTAINERID=..., CNI_NETNS=...
//                  stdin: <plugin 配置 + prevResult>
//                  stdout: <Result JSON>
//   下一个插件用上一个的 Result 作为 prevResult 继续。
//
// 想看真实实现（netlink veth + netns + iptables masquerade）：
//   https://github.com/containernetworking/plugins/blob/main/plugins/main/bridge/bridge.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// 支持的 spec 版本——按 CNI v1.0 / v1.1 标准列出。
var supportedVersions = []string{"0.3.0", "0.3.1", "0.4.0", "1.0.0", "1.1.0"}

// PluginConf 是 stdin 接收的 JSON。conflist 里 plugins 数组的当前对象
// 加上 chaining 时上一个插件透传过来的 prevResult。
type PluginConf struct {
	CNIVersion string          `json:"cniVersion"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Bridge     string          `json:"bridge,omitempty"`
	IPAM       *IPAMConf       `json:"ipam,omitempty"`
	PrevResult json.RawMessage `json:"prevResult,omitempty"`
}

type IPAMConf struct {
	Type   string `json:"type"`
	Subnet string `json:"subnet,omitempty"`
}

// Result 是 ADD / CHECK 成功时 stdout 的 JSON 输出，按 spec v1.0。
type Result struct {
	CNIVersion string      `json:"cniVersion"`
	Interfaces []Interface `json:"interfaces,omitempty"`
	IPs        []IPConfig  `json:"ips,omitempty"`
	Routes     []Route     `json:"routes,omitempty"`
	DNS        DNS         `json:"dns,omitempty"`
}

type Interface struct {
	Name    string `json:"name"`
	Mac     string `json:"mac,omitempty"`
	Sandbox string `json:"sandbox,omitempty"`
}

// 注意 Interface 是 *int 而不是 int：spec 要求"未设置"和"指向 index 0"必须可区分。
type IPConfig struct {
	Address   string `json:"address"`
	Gateway   string `json:"gateway,omitempty"`
	Interface *int   `json:"interface,omitempty"`
}

type Route struct {
	Dst string `json:"dst"`
	GW  string `json:"gw,omitempty"`
}

type DNS struct {
	Nameservers []string `json:"nameservers,omitempty"`
	Search      []string `json:"search,omitempty"`
}

// ErrorReply 按 spec 规定写到 stdout（即使失败也要 JSON 化输出）。
// code 1-99 是 spec 保留：1=incompatible CNI version, 7=invalid configuration,
// 11=try again later, 等等。100+ 可由插件自定义。
type ErrorReply struct {
	CNIVersion string `json:"cniVersion"`
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Details    string `json:"details,omitempty"`
}

// VersionReply 是 CNI_COMMAND=VERSION 的输出格式。
type VersionReply struct {
	CNIVersion        string   `json:"cniVersion"`
	SupportedVersions []string `json:"supportedVersions"`
}

func main() {
	cmd := os.Getenv("CNI_COMMAND")

	// VERSION 不读 stdin
	if cmd == "VERSION" {
		printJSON(&VersionReply{CNIVersion: "1.0.0", SupportedVersions: supportedVersions})
		return
	}

	conf, err := loadConf()
	if err != nil {
		bail(7, "invalid configuration", err.Error())
	}

	switch cmd {
	case "ADD":
		cmdAdd(conf)
	case "DEL":
		cmdDel(conf)
	case "CHECK":
		cmdCheck(conf)
	case "GC":
		// v1.1 新增：清理孤儿 attachment。本 demo 无状态，直接成功。
		return
	case "STATUS":
		// v1.1 新增：健康检查。本 demo 无状态，直接成功。
		return
	default:
		bail(4, "unknown CNI_COMMAND", cmd)
	}
}

func loadConf() (*PluginConf, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty stdin")
	}
	conf := &PluginConf{}
	if err := json.Unmarshal(data, conf); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if conf.CNIVersion == "" {
		return nil, fmt.Errorf("missing cniVersion")
	}
	return conf, nil
}

func cmdAdd(conf *PluginConf) {
	netns := os.Getenv("CNI_NETNS")
	ifName := os.Getenv("CNI_IFNAME")
	contID := os.Getenv("CNI_CONTAINERID")

	// hostNetwork Pod 没有自己的 netns，runtime 会传空串——这里按 spec 直接成功。
	if netns == "" {
		fmt.Fprintln(os.Stderr, "cni-bridge: empty CNI_NETNS, skipping (likely hostNetwork pod)")
		printJSON(&Result{CNIVersion: conf.CNIVersion})
		return
	}

	// 生产实现里这一步会：
	//   1. netlink.LinkAdd 创建 bridge（若不存在）
	//   2. netlink.LinkAdd 创建 veth pair
	//   3. netlink.LinkSetMaster 把 host 端接入 bridge
	//   4. netlink.LinkSetNsFd 把 cont 端 move 进 netns
	//   5. 调 IPAM 子插件 fork+exec /opt/cni/bin/host-local 拿真实 IP
	//   6. 进 netns（setns(2)）配 IP / 路由
	// 这里只用一个 dummy 假装成功。
	ifIdx := 2
	fakeIP := fmt.Sprintf("10.244.0.%d/24", 10+(len(contID)%240))
	result := &Result{
		CNIVersion: conf.CNIVersion,
		Interfaces: []Interface{
			{Name: bridgeName(conf), Mac: "0a:58:0a:f4:00:01"},
			{Name: "vethDEMO", Mac: "0a:58:0a:f4:00:02"},
			{Name: ifName, Mac: "0a:58:0a:f4:00:03", Sandbox: netns},
		},
		IPs: []IPConfig{
			{Address: fakeIP, Gateway: "10.244.0.1", Interface: &ifIdx},
		},
		Routes: []Route{
			{Dst: "0.0.0.0/0", GW: "10.244.0.1"},
		},
		DNS: DNS{
			Nameservers: []string{"10.96.0.10"},
			Search:      []string{"cluster.local", "svc.cluster.local"},
		},
	}
	printJSON(result)
}

func cmdDel(conf *PluginConf) {
	// 真实实现：
	//   1. 进 netns 把 ifName 删掉（其实 netns 销毁会自动清理 veth）
	//   2. 调 IPAM ExecDel 释放 lease 文件
	//   3. 清理 host 上的 iptables 规则
	// 必须幂等：资源不存在不算错。runtime 可能多次调 DEL。
	// 本 demo 无状态，直接成功。
}

func cmdCheck(conf *PluginConf) {
	// 真实实现：进 netns 检查 ifName 存在、IP 配置与 prevResult 一致。
	// 本 demo 无状态，直接成功。
}

func bridgeName(conf *PluginConf) string {
	if conf.Bridge != "" {
		return conf.Bridge
	}
	return "cni0"
}

func printJSON(v interface{}) {
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "cni-bridge: encode result: %v\n", err)
		os.Exit(1)
	}
}

func bail(code int, msg, details string) {
	_ = json.NewEncoder(os.Stdout).Encode(&ErrorReply{
		CNIVersion: "1.0.0",
		Code:       code,
		Msg:        msg,
		Details:    details,
	})
	os.Exit(1)
}
