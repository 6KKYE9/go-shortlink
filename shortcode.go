package main

import (
	"crypto/rand"
	"math/big"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// genCode 生成一个长度为 n 的 base62 短码。用 crypto/rand 避免可预测的顺序。
func genCode(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(base62Chars)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			// 基本不可能走到，rand.Reader 失败就退化成固定字符。
			b[i] = base62Chars[0]
			continue
		}
		b[i] = base62Chars[idx.Int64()]
	}
	return string(b)
}
