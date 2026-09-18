# mychain

用 Go 实现的一个简单区块链，带工作量证明（PoW）和 HTTP 接口。

## 功能

- SHA-256 计算区块哈希，通过前导零数量控制难度
- 工作量证明挖矿（`Mine`）
- 链上数据持久化到 `mychain.json`
- 启动时校验链的完整性，非法则拒绝启动
- 并发安全（`sync.Mutex`），支持并发挖矿

## 运行

```bash
go run main.go
```

默认监听 `:8080`。首次启动会自动创建创世区块（难度 5）并写入 `mychain.json`。

## 接口

### 查看全部区块

```bash
curl http://localhost:8080/blocks
```

### 挖矿（添加新区块）

```bash
curl -X POST http://localhost:8080/mine \
  -H 'Content-Type: application/json' \
  -d '{"data":["hello"]}'
```

### 校验链是否有效

```bash
curl http://localhost:8080/valid
# {"valid":true}
```

## 说明

- 链数据保存在 `mychain.json`（已在 `.gitignore` 中忽略），删除该文件即可重新生成。
- 区块哈希 = `sha256(prev_block_hash|timestamp|data|nonce)`。
