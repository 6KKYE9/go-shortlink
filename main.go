package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "监听地址")
	codeLen := flag.Int("len", 6, "短码长度")
	baseURL := flag.String("base", "http://127.0.0.1:8080", "返回的短链接前缀")
	nodes := flag.String("nodes", "node1,node2,node3", "分布式节点列表，逗号分隔（演示用）")
	flag.Parse()

	store := newMemStore()
	srv := newServer(store, *codeLen, *baseURL)

	// 一致性哈希环：短码进来先算落在哪个节点。练手阶段所有节点共用一个内存 store，
	// 真部署时每个节点连自己的分片存储即可，路由逻辑不用改。
	nodeList := splitNodes(*nodes)
	ring := newHashRing(100, nodeList...)
	log.Printf("短链接服务启动 %s，一致性哈希环节点数: %d", *addr, len(nodeList))
	log.Printf("示例: 短码 'abc123' 将路由到节点 %q", ring.get("abc123"))

	mux := http.NewServeMux()
	srv.Register(mux)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func splitNodes(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
