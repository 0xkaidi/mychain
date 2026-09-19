# mychain

用 Go 实现的一个简单区块链：支持转账交易、待打包交易池、工作量证明（PoW）和 HTTP 接口。

## 功能

- SHA-256 计算区块哈希，通过前导零数量控制难度（默认 5）
- 工作量证明挖矿（`Mine`）
- 交易模型：每笔交易是 `{from, to, amount}`，区块打包一批交易
- 待打包交易池：交易先提交到 `Pending`，挖矿时整批打包
- 提交交易时校验：`from` 不能为空、余额充足（含未确认交易）
- 余额查询，支持把未确认交易计入余额
- 链数据持久化到 `mychain.json`，启动时校验链的完整性，非法则拒绝启动
- 并发安全（`sync.Mutex`），落盘失败会回滚区块并把交易退回交易池

## 目录结构

```
main.go                  加载链、启动 HTTP 服务
internal/chain/block.go  区块结构、哈希计算、挖矿
internal/chain/chain.go  链逻辑：交易池、提交交易、挖矿、余额、校验
internal/chain/store.go  读写 mychain.json
internal/api/handler.go  HTTP 接口
```

## 运行

```bash
go run main.go
```

默认监听 `:8080`。首次启动会自动创建创世区块（难度 5，含一笔 `"" → genesis` 的 1000 初始交易）并写入 `mychain.json`。

## 接口

### 提交交易

```bash
curl -X POST http://localhost:8080/transaction \
  -H 'Content-Type: application/json' \
  -d '{"from":"genesis","to":"alice","amount":300}'
# {"status":"accepted"}
```

余额不足或 `from` 为空会返回 400，例如 `insufficient balance`。

### 挖矿（打包交易池中的交易）

```bash
curl -X POST http://localhost:8080/mine
```

没有待打包交易时返回 400 `mine failed`。

### 查询余额

```bash
curl http://localhost:8080/balance/alice
# {"balance":300}
```

余额包含交易池中尚未确认的交易。

### 查看全部区块

```bash
curl http://localhost:8080/blocks
```

### 校验链是否有效

```bash
curl http://localhost:8080/valid
# {"valid":true}
```

## 说明

- 链数据保存在 `mychain.json`（已在 `.gitignore` 中忽略），删除该文件即可重新生成。
- 交易池只存在内存中，重启后未打包的交易会丢失（已确认的区块不受影响）。
- 区块哈希 = `sha256(prev_block_hash|timestamp|transactions|nonce)`，其中 `transactions` 为交易的 JSON 编码。
