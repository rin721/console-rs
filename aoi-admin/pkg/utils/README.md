# pkg/utils - 通用工具集

## 概述

Utils 是一个通用工具库，提供常用的工具函数和组件，包括分布式 ID 生成、IP 地址验证、设备指纹生成、端口查找等功能。

## API 分类

- 定位：[CONFIRMED] 内部支撑工具包。
- 稳定边界：当前供 `internal/*` 和少量 `types/*` 使用的 ID、地址、端口、设备 ID、i18n helper。
- 当前风险：[RISK] 包内能力较杂，默认 Snowflake 实例 panic 策略需要后续确认。
- 非目标：[CONFIRMED] 本包暂不承诺外部兼容；新增工具应先判断是否应进入更具体的包。

### 特性

- ✅ **Snowflake ID 生成器** - 分布式唯一 ID 生成
- ✅ **IP 地址验证** - HTTP 监听地址合法性验证
- ✅ **设备 ID 生成** - 基于硬件信息的设备指纹
- ✅ **端口查找** - 自动查找可用端口
- ✅ **线程安全** - 所有工具都是并发安全的
- ✅ **零依赖** - 工具函数独立，按需使用

## 工具列表

| 工具                | 文件                    | 说明                          |
| ------------------- | ----------------------- | ----------------------------- |
| Snowflake ID 生成器 | `snowflake.go`          | 分布式唯一 ID 生成            |
| IP 地址验证         | `ip.go`                 | HTTP 监听地址验证             |
| 设备 ID 生成        | `drive_id.go`           | 生成设备唯一标识              |
| 端口查找            | `get_available_port.go` | 查找指定范围内的可用 TCP 端口 |

## 安装

```bash
# Snowflake ID 生成器依赖
go get github.com/bwmarrin/snowflake
```

## 快速开始

### 1. Snowflake ID 生成器

生成分布式唯一 ID，适用于数据库主键、用户 ID、订单号等场景。

```go
import "github.com/rei0721/go-scaffold/pkg/utils"

// 创建 ID 生成器（单机环境）
gen := utils.DefaultSnowflake()

// 生成 int64 ID
id := gen.NextID()
fmt.Println(id) // 1747234567890123456

// 生成字符串 ID（适合前端使用）
idStr := gen.NextIDString()
fmt.Println(idStr) // "1747234567890123456"
```

**分布式环境**：

```go
// 每个实例使用不同的 nodeID（0-1023）
nodeID := getNodeIDFromConfig() // 例如：从配置文件读取
gen, err := utils.NewSnowflake(nodeID)
if err != nil {
    log.Fatal(err)
}

id := gen.NextID()
```

### 2. IP 地址验证

验证 HTTP 监听地址是否合法且可被绑定。

```go
import "github.com/rei0721/go-scaffold/pkg/utils"

// 验证监听地址
err := utils.IsValidListenAddr(":8080")
if err != nil {
    log.Fatal("invalid listen address:", err)
}

// 更严格的验证（实际尝试绑定）
err = utils.IsValidHTTPListenAddr("127.0.0.1:8080")
if err != nil {
    log.Fatal("cannot bind to address:", err)
}
```

### 3. 设备 ID 生成

生成基于硬件信息的设备唯一标识。

```go
import "github.com/rei0721/go-scaffold/pkg/utils"

// 生成设备 ID
appSalt := "my-app-v1.0" // 应用盐值，一旦发布不要修改
deviceID := utils.GenerateDeviceID(appSalt)
fmt.Println(deviceID)
// 输出: "a1b2c3d4e5f6..."（64位十六进制字符串）
```

### 4. 端口查找

在指定范围内查找可用的 TCP 端口。

```go
import "github.com/rei0721/go-scaffold/pkg/utils"

// 查找可用端口（30000-40000）
port, err := utils.GetAvailablePort(30000, 40000)
if err != nil {
    log.Fatal("no available port:", err)
}
fmt.Println("Available port:", port)

// 排除某些端口
port, err = utils.GetAvailablePort(30000, 40000, 30001, 30002)
```

## API 文档

### Snowflake ID 生成器

#### IDGenerator 接口

```go
type IDGenerator interface {
    NextID() int64
    NextIDString() string
}
```

#### NewSnowflake

创建 Snowflake ID 生成器。

```go
func NewSnowflake(nodeID int64) (IDGenerator, error)
```

**参数**：

- `nodeID` (int64) - 节点 ID，取值范围 0-1023

**返回**：

- `IDGenerator` - ID 生成器接口
- `error` - 创建失败时的错误

**注意事项**：

- 相同 nodeID 的实例会生成冲突的 ID
- 时钟回拨会导致 ID 重复，生产环境需使用 NTP 同步时间
- 确保分布式环境中每个实例使用不同的 nodeID

**示例**：

```go
// 单机环境
gen, err := utils.NewSnowflake(1)

// 分布式环境（从配置读取）
gen, err := utils.NewSnowflake(config.NodeID)

// K8s 环境（从 Pod 序号）
podIndex := getPodIndex() // 0-1023
gen, err := utils.NewSnowflake(int64(podIndex))
```

#### DefaultSnowflake

创建默认的 ID 生成器（nodeID=1）。

```go
func DefaultSnowflake() IDGenerator
```

**适用场景**：

- 开发和测试环境
- 单机部署
- 不需要考虑分布式的场景

**不适用场景**：

- 多实例部署（会生成重复 ID）
- 需要自定义 nodeID 的场景

#### NextID

生成 int64 类型的唯一 ID。

```go
func (g *IDGenerator) NextID() int64
```

**返回**：

- `int64` - 唯一 ID，按时间递增

**使用场景**：

- 数据库主键
- 用户 ID
- 订单号

**线程安全**：可以在多个 goroutine 中并发调用

#### NextIDString

生成字符串类型的唯一 ID。

```go
func (g *IDGenerator) NextIDString() string
```

**返回**：

- `string` - 十进制字符串格式的唯一 ID

**使用场景**：

- TraceID（请求追踪 ID）
- 前端需要的 ID（避免 JavaScript 精度问题）
- JSON 序列化

**注意**：JavaScript Number 只能精确表示 53 位整数，超过此范围建议使用字符串格式。

### IP 地址验证

#### IsValidListenAddr

验证监听地址是否合法且可被 HTTP 服务器绑定。

```go
func IsValidListenAddr(addr string) error
```

**参数**：

- `addr` (string) - 监听地址，格式：`host:port`

**返回**：

- `error` - 验证失败时的错误，成功返回 nil

**允许的地址格式**：

- `:8080` - 监听所有网卡
- `0.0.0.0:8080` - 监听所有 IPv4
- `127.0.0.1:8080` - 本地回环
- `localhost:8080` - 本地主机
- `[::]:8080` - 所有 IPv6
- 本机网卡 IP + 端口

**禁止的地址**：

- 公网 IP
- 非本机 IP
- 非法 host 或端口

**示例**：

```go
// ✅ 合法
err := utils.IsValidListenAddr(":8080")
err := utils.IsValidListenAddr("127.0.0.1:8080")

// ❌ 非法
err := utils.IsValidListenAddr("1.2.3.4:8080") // 非本机 IP
err := utils.IsValidListenAddr("invalid")      // 格式错误
```

#### IsValidHTTPListenAddr

通过实际绑定测试地址是否可用。

```go
func IsValidHTTPListenAddr(addr string) error
```

**参数**：

- `addr` (string) - 监听地址

**返回**：

- `error` - 验证失败时的错误

**核心原则**：

- 只要 `net.Listen("tcp", addr)` 能成功，就认为合法
- 能精准拦截非法地址

**示例**：

```go
err := utils.IsValidHTTPListenAddr("127.0.0.1:8080")
if err != nil {
    // 地址无法绑定
}
```

### 设备 ID 生成

#### GenerateDeviceID

生成设备唯一标识符。

```go
func GenerateDeviceID(appSalt string) string
```

**参数**：

- `appSalt` (string) - 应用盐值，用于防止同一台机器在不同软件中生成相同设备码

**返回**：

- `string` - 64 位十六进制字符串（SHA256 哈希值）

**生成原理**：

基于以下硬件信息生成指纹：

1. 操作系统信息（GOOS、GOARCH）
2. 主机名
3. 真实网卡 MAC 地址（过滤虚拟网卡）
4. 应用盐值

**注意事项**：

- `appSalt` 一旦发布不要轻易修改
- 虚拟网卡会被自动过滤
- 适合软件授权、设备绑定等场景

**示例**：

```go
// 生成设备 ID
deviceID := utils.GenerateDeviceID("my-app-v1.0")

// 验证设备
storedDeviceID := getStoredDeviceID()
if deviceID != storedDeviceID {
    return errors.New("device mismatch")
}
```

### 端口查找

#### GetAvailablePort

在指定范围内查找可用的 TCP 端口。

```go
func GetAvailablePort(start, end int, exclude ...int) (int, error)
```

**参数**：

- `start` (int) - 起始端口
- `end` (int) - 结束端口
- `exclude` (...int) - 需要排除的端口（可选）

**返回**：

- `int` - 可用端口
- `error` - 找不到可用端口时的错误

**验证方式**：

- 通过实际绑定 `net.Listen("tcp", "0.0.0.0:port")` 测试端口是否可用
- 使用互斥锁防止并发抢占同一端口

**示例**：

```go
// 查找可用端口
port, err := utils.GetAvailablePort(30000, 40000)
if err != nil {
    log.Fatal("no available port")
}

// 排除某些端口
port, err := utils.GetAvailablePort(30000, 40000, 30001, 30002)

// 在找到的端口上启动服务
server.Run(fmt.Sprintf(":%d", port))
```

## 使用场景

### 场景 1: 分布式 ID 生成

```go
// 初始化 ID 生成器
var idGen utils.IDGenerator

func init() {
    nodeID := getNodeID() // 从配置或环境变量获取
    var err error
    idGen, err = utils.NewSnowflake(nodeID)
    if err != nil {
        panic(err)
    }
}

// 创建用户
func createUser(name string) (*User, error) {
    user := &User{
        ID:   idGen.NextID(),
        Name: name,
    }
    return user, db.Create(user).Error
}

// 生成 TraceID
func generateTraceID() string {
    return idGen.NextIDString()
}
```

### 场景 2: 动态端口分配

```go
// 开发环境自动查找可用端口
func startDevServer() error {
    port, err := utils.GetAvailablePort(8000, 9000)
    if err != nil {
        return err
    }

    log.Printf("Server starting on port %d", port)
    return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
}

// 微服务动态端口
func registerService() error {
    port, err := utils.GetAvailablePort(30000, 40000)
    if err != nil {
        return err
    }

    // 向注册中心注册服务
    return serviceRegistry.Register("my-service", fmt.Sprintf(":%d", port))
}
```

### 场景 3: 软件授权验证

```go
// 生成机器码
func generateMachineCode() string {
    return utils.GenerateDeviceID("my-software-v2.0")
}

// 验证授权
func validateLicense(licenseKey string) error {
    machineCode := generateMachineCode()

    // 从服务器验证授权
    valid, err := licenseServer.Validate(licenseKey, machineCode)
    if err != nil || !valid {
        return errors.New("invalid license")
    }

    return nil
}
```

### 场景 4: HTTP 服务器配置验证

```go
// 启动前验证配置
func startServer(config *Config) error {
    // 验证监听地址
    if err := utils.IsValidListenAddr(config.ListenAddr); err != nil {
        return fmt.Errorf("invalid listen address: %w", err)
    }

    // 启动服务器
    return http.ListenAndServe(config.ListenAddr, handler)
}
```

## 最佳实践

### 1. Snowflake ID 生成器

**单例模式**：

```go
var (
    idGenOnce sync.Once
    idGen     utils.IDGenerator
)

func GetIDGenerator() utils.IDGenerator {
    idGenOnce.Do(func() {
        gen, err := utils.NewSnowflake(getNodeID())
        if err != nil {
            panic(err)
        }
        idGen = gen
    })
    return idGen
}
```

**分布式环境配置 nodeID**：

```go
// 方式1: 从配置文件
nodeID := viper.GetInt64("node_id")

// 方式2: 从环境变量
nodeID, _ := strconv.ParseInt(os.Getenv("NODE_ID"), 10, 64)

// 方式3: 基于 IP 自动生成（开发环境）
nodeID := getNodeIDFromIP()

// 方式4: K8s StatefulSet（推荐）
// Pod 名称：my-app-0, my-app-1, my-app-2
podName := os.Getenv("HOSTNAME") // my-app-1
index := extractIndex(podName)    // 1
nodeID := int64(index)
```

### 2. 设备 ID 生成

**固定应用盐值**：

```go
const AppSalt = "my-app-v1.0.0" // 发布后不要修改

func init() {
    deviceID := utils.GenerateDeviceID(AppSalt)
    storeDeviceID(deviceID)
}
```

**不要频繁生成**：

```go
// ✅ 好的做法：缓存设备 ID
var cachedDeviceID string

func GetDeviceID() string {
    if cachedDeviceID == "" {
        cachedDeviceID = utils.GenerateDeviceID(AppSalt)
    }
    return cachedDeviceID
}

// ❌ 不好的做法：每次都生成
func GetDeviceID() string {
    return utils.GenerateDeviceID(AppSalt) // 浪费性能
}
```

### 3. 端口查找

**合理设置端口范围**：

| 用途     | 推荐范围    | 说明                     |
| -------- | ----------- | ------------------------ |
| 开发环境 | 8000-9000   | 常用开发端口             |
| 测试环境 | 10000-20000 | 避免与开发环境冲突       |
| 生产环境 | 固定端口    | 不建议动态分配           |
| 微服务   | 30000-40000 | Kubernetes NodePort 范围 |

**错误处理**：

```go
port, err := utils.GetAvailablePort(start, end)
if err != nil {
    // 降级策略：使用默认端口
    port = defaultPort
    log.Warn("no available port, using default", "port", port)
}
```

## 性能考虑

### Snowflake ID 生成器

- **性能**：每毫秒可生成 4096 个 ID
- **时间精度**：毫秒级
- **有效期**：约 69 年（从 Epoch 开始计算）
- **并发安全**：内置互斥锁，性能优异

### 设备 ID 生成

- **计算成本**：SHA256 哈希，性能较好
- **建议**：应用启动时生成一次并缓存
- **缓存时长**：整个应用生命周期

### 端口查找

- **查找方式**：顺序尝试绑定
- **性能影响**：取决于端口范围大小
- **优化建议**：缩小查找范围，排除已知占用端口

## 项目结构

```
pkg/utils/
├── snowflake.go           # Snowflake ID 生成器
├── ip.go                  # IP 地址验证
├── drive_id.go            # 设备 ID 生成
├── get_available_port.go  # 端口查找
└── README.md              # 本文档
```

## 依赖

### 必须依赖

- `github.com/bwmarrin/snowflake` - Snowflake ID 生成算法实现

### 标准库依赖

- `net` - 网络操作
- `crypto/sha256` - SHA256 哈希
- `runtime` - 运行时信息
- `os` - 操作系统信息

## 参考链接

- [Snowflake ID 算法](https://github.com/twitter-archive/snowflake)
- [bwmarrin/snowflake](https://github.com/bwmarrin/snowflake)
- [设备指纹技术](https://en.wikipedia.org/wiki/Device_fingerprint)
