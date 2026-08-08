package main

import (
	"crypto/sha1"
	"fmt"
	"sort"
)

// hashRing 是一致性哈希环，用来在多个短链接节点之间分配短码。
// 为什么需要它：短链接服务通常多机部署，同一个短码应该稳定地落到同一台机器
// （或者同一份分片存储）上，这样读的时候不用广播所有节点。加机器时只有
// 1/N 的短码需要迁移，而不是全部重算。
type hashRing struct {
	// 每个真实节点复制成若干个虚拟节点，打散到环上，让分布更均匀。
	replicas int
	ring     map[uint32]string // 哈希值 -> 节点名
	sorted   []uint32          // 环上所有哈希值，排序好做二分查找
}

func newHashRing(replicas int, nodes ...string) *hashRing {
	h := &hashRing{replicas: replicas, ring: make(map[uint32]string)}
	h.add(nodes...)
	return h
}

func (h *hashRing) add(nodes ...string) {
	for _, n := range nodes {
		for i := 0; i < h.replicas; i++ {
			key := fmt.Sprintf("%s#%d", n, i)
			hv := hashKey(key)
			h.ring[hv] = n
			h.sorted = append(h.sorted, hv)
		}
	}
	sort.Slice(h.sorted, func(i, j int) bool { return h.sorted[i] < h.sorted[j] })
}

// get 给一个短码算出它该落在哪个节点。
func (h *hashRing) get(key string) string {
	if len(h.sorted) == 0 {
		return ""
	}
	hv := hashKey(key)
	// 在环上找第一个 >= hv 的虚拟节点，没有就绕回最小的。
	idx := sort.Search(len(h.sorted), func(i int) bool { return h.sorted[i] >= hv })
	if idx == len(h.sorted) {
		idx = 0
	}
	return h.ring[h.sorted[idx]]
}

func hashKey(s string) uint32 {
	sum := sha1.Sum([]byte(s))
	// 取前 4 字节当 uint32。
	return uint32(sum[0])<<24 | uint32(sum[1])<<16 | uint32(sum[2])<<8 | uint32(sum[3])
}
