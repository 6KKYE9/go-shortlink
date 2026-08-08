# go-shortlink

一个短链接服务：POST 一个长 URL，换回一个短码；拿短码 GET 一下，302 跳回原地址。练手版用内存存储，但存储层抽成了接口，想接 Redis/MySQL 直接换实现，HTTP 层一行不用动。

## 跑起来

```powershell
go run .
```

缩短：

```powershell
curl -X POST http://127.0.0.1:8080/api/shorten `
  -H "Content-Type: application/json" `
  -d '{"url":"https://example.com/some/very/long/path"}'
# 返回 {"code":"aZ3x9Q","short":"http://127.0.0.1:8080/aZ3x9Q","url":"..."}
```

跳转：浏览器直接开 `http://127.0.0.1:8080/aZ3x9Q` 就跳走了。

常用参数：

```powershell
go run . -addr :8080 -len 6 -base http://short.example.com -nodes node1,node2,node3
```

## 分布式这块怎么做的

短码用 base62（数字+大小写字母）随机生成，碰撞就重来。重点在 `hashRing`：一致性哈希环。

多机部署时，同一个短码得稳定落到同一台机器（或同一份分片），读的时候才不用广播所有节点。一致性哈希的做法是：

- 每个真实节点复制成 100 个虚拟节点，sha1 打散到 0~2^32 的环上
- 一个短码算哈希，在环上找顺时针第一个虚拟节点，就是它该去的节点
- 加一台机器，只有约 1/N 的短码需要迁移，其余不动

练手阶段所有节点共用一份内存 store，所以路由只是演示；真上线把每个节点的 store 换成各自的分片存储，路由逻辑不用改。

## 没做的事

- 内存存储重启就没了，没落盘也没接 Redis
- 没防滥用：谁都能一直发，没限流没鉴权
- 短码没做自定义/过期时间

## 测试

```powershell
go test ./...
```

缩短→跳转的完整链路用 httptest 真起服务测了，一致性哈希的稳定性、分布均匀度、store 未命中也都有用例。
