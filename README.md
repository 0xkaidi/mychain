# mychain

用 Go 实现的一个简单区块链：支持 ECDSA 签名的转账交易、钱包、待打包交易池、工作量证明（PoW）、HTTP 接口和命令行工具。

## 功能

- 钱包：ECDSA P-256 密钥对，私钥以 PEM 保存，地址 = `sha256(未压缩公钥字节)`
- 交易签名与验签：交易体（from/to/amount/pubkey）经 SHA-256 后用 ECDSA 签名
- 提交交易时校验：`from`/`pub_key`/`signature` 非空、公钥十六进制可解析且与 `from` 地址匹配、签名有效、余额充足（含未确认交易）
- 待打包交易池：交易先提交到 `Pending`，挖矿时整批打包
- 挖矿奖励：每个区块附带一笔 50 的 coinbase 交易给矿工
- SHA-256 计算区块哈希，通过前导零数量控制难度（固定为 5）
- 余额查询，支持把未确认交易计入余额
- 链数据持久化到 `mychain-<port>.json`（每个端口一份），启动时完整校验：创世块、区块哈希与难度、每块恰好一笔 50 的 coinbase、每笔普通交易的地址/签名/余额，非法则拒绝启动
- 并发安全（`sync.Mutex`），落盘失败会回滚区块并把交易退回交易池
- 基础节点同步：每 10 秒从邻居拉取完整链，挖矿成功后广播完整链；仅接受更长且校验有效的链
- 命令行工具 `cmd/cli`，通过 HTTP 调用节点接口

## 目录结构

```
main.go                     加载链、启动 HTTP 服务
cmd/cli/main.go             命令行工具
internal/chain/block.go     区块结构、哈希计算、挖矿
internal/chain/chain.go     链逻辑：交易池、提交交易、挖矿、余额、校验
internal/chain/transaction.go  交易结构与签名原文
internal/chain/wallet.go    钱包、地址、PEM 读写
internal/chain/crypto.go    ECDSA 签名/验签
internal/chain/store.go     读写链文件
internal/api/handler.go     HTTP 接口（含 /sync）
internal/p2p/sync.go        邻居节点、定时拉取与广播
```

## 运行

启动节点：

```bash
go run .
```

默认监听 `:8080`，链文件为 `mychain-<port>.json`（默认即 `mychain-8080.json`）。首次启动会自动创建创世区块（难度 5，含一笔 `"" → genesis` 的 1000 初始交易）并写入该文件。

可用参数：

```bash
go run . -port 8081                            # 换端口，链文件为 mychain-8081.json
go run . -peers http://localhost:8081,http://localhost:8082  # 多个邻居用逗号分隔
```

`-peers` 支持 `http://host:port`，省略 `http://` 时会自动补全。不同端口使用独立的链文件，通过邻居同步更长的有效链；待打包交易暂不广播。

### 启动两个节点

分别在两个终端中运行（保持终端开启）：

```bash
# 终端 1
go run . -port 8080 -peers http://localhost:8081

# 终端 2
go run . -port 8081 -peers http://localhost:8080
```

启动后程序应持续运行，不会立即返回命令提示符。HTTP 服务在主 goroutine 中阻塞，定时同步在后台运行。用第三个终端检查：

```bash
curl http://localhost:8080/blocks
curl http://localhost:8081/blocks
```

节点每 10 秒拉取邻居的 `/blocks`，再提交给自身 `/sync`；通过 `/mine` 挖矿成功后也会向邻居广播完整链。

### 同步日志说明

`post to http://localhost:8080/sync : rejected` 表示本次链替换被拒绝，不代表启动失败。`/sync` 的 JSON 响应中包含 `reason`，但当前后台日志只打印 `status`：

- `new chain is not longer`：对方链不比本地长（包括等长），无需替换，属于正常情况。
- `new chain is invalid`：候选链未通过校验，需要排查数据。
- 其他原因可能来自链文件保存失败。

因此不能仅凭 `rejected` 判断链无效。相同长度的分叉目前不会自动解决；邻居尚未启动时出现连接失败，可在邻居启动后等待下一轮同步。

编译（含 CLI）：

```bash
go build -o mychain .             # 节点
go build -o mychain-cli ./cmd/cli # 命令行工具
```

## 快速开始

```bash
# 1. 生成两个钱包（默认写到 ./key.pem，可用 -out 指定）
./mychain-cli keygen -out alice.pem
./mychain-cli keygen -out bob.pem

# 2. 查看地址（后面用得到）
./mychain-cli inspect -key alice.pem   # address: ... / pubkey: ...

# 3. 用 alice 挖矿，获得 50 奖励
./mychain-cli mine -key alice.pem

# 4. alice 转给 bob（自动签名）
./mychain-cli send -key alice.pem <bob地址> 30

# 5. 再挖一次把交易打包确认
./mychain-cli mine -key alice.pem

# 6. 查询余额
./mychain-cli balance <alice地址>
./mychain-cli balance <bob地址>
```

## 命令行工具

CLI 需要节点已在 `http://localhost:8080` 运行，输出为接口返回的原始 JSON，出错时打印原因并以状态码 1 退出。
`send` / `mine` / `inspect` 用 `-key` 指定私钥，默认为 `./key.pem`；`keygen` 用 `-out` 指定输出路径。

```bash
./mychain-cli keygen [-out key.pem]        # 生成密钥对，打印地址
./mychain-cli inspect [-key key.pem]       # 查看私钥对应的地址和公钥
./mychain-cli send [-key key.pem] <to> <amount>  # 用该私钥签名并提交交易
./mychain-cli mine [-key key.pem]          # 挖矿，奖励给该私钥的地址
./mychain-cli balance <address>            # 查询余额（含未确认交易）
./mychain-cli blocks                       # 查看全部区块
./mychain-cli valid                        # 校验链是否有效
```

## 接口

### 提交交易

交易需要用 `from` 对应的私钥签名，签名原文为 `from|to|amount|pub_key`。

```bash
curl -X POST http://localhost:8080/transaction \
  -H 'Content-Type: application/json' \
  -d '{"from":"<地址>","to":"<地址>","amount":30,"pub_key":"<公钥>","signature":"<签名>"}'
# {"status":"accepted"}
```

校验失败会返回 400，例如 `amount should > 0`、`insufficient balance`、`pub key and address does not match`、
`signature cant be empty`、`verify failed: invalid signature`。

### 挖矿（打包交易池中的交易）

```bash
curl -X POST http://localhost:8080/mine \
  -H 'Content-Type: application/json' \
  -d '{"miner":"<接收奖励的地址>"}'
```

交易池为空时仍会出块（只含 coinbase 奖励），请求体缺失或不是合法 JSON 返回 400 `invalid json`。

### 查询余额

```bash
curl http://localhost:8080/balance/<address>
# {"balance":80}
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

### 同步链

`POST /sync` 接收 JSON 对象，供节点间同步使用：

- `blocks`：完整候选链，结构与 `/blocks` 返回的区块对象数组一致。
- `difficulty`：候选链难度，目前为 `5`。

仅当候选链比本地更长且校验通过时才替换并保存。

- 接受：`{"status":"accepted"}`
- 拒绝：`{"status":"rejected","reason":"new chain is not longer"}` 等

接受和业务拒绝均返回 HTTP 200，调用方需要检查 `status`；JSON 解析失败返回 400，非 POST 请求返回 405。

## 说明

- 链数据保存在 `mychain-<port>.json`，私钥为 `*.pem`，两者都已在 `.gitignore` 中忽略。
- 链文件名带端口号：旧版的 `mychain.json` 不会再被读取，需要用时手动改名为 `mychain-8080.json`。
- 交易池只存在内存中，重启后未打包的交易会丢失（已确认的区块不受影响）。
- 区块哈希 = `sha256(prev_block_hash|timestamp|transactions|nonce)`，其中 `transactions` 为交易的 JSON 编码。
- 创世区块中 1000 的初始余额记在字面地址 `genesis` 名下，没有对应私钥，因此无法花掉；新币只能通过挖矿奖励产生。
- 地址是对未压缩 SEC1 公钥字节做 `sha256` 后的十六进制；节点用它校验交易的 `from` 与 `pub_key` 是否匹配。
- 难度是常量 `chain.Difficulty = 5`，`mychain-<port>.json` 中记录的难度与之不一致时链会被判为非法。
- CLI 的服务端地址目前硬编码在 `cmd/cli/main.go` 的 `baseURL`。
