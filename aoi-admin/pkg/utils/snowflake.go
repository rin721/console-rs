// Package utils 提供使用 Snowflake 算法的唯一 ID 生成
// Snowflake 是 Twitter 开源的分布式 ID 生成算法
// 优点:
// - 生成的 ID 全局唯一
// - 按时间递增,有利于数据库索引
// - 不依赖数据库,性能高
// - 可以在应用层生成,无需额外服务
package utils

// 本文件属于通用工具层，提供无业务状态的地址校验、端口选择、设备标识或国际化代理能力。

import (
	"strconv"

	"github.com/bwmarrin/snowflake"
)

// IDGenerator 定义 ID 生成的接口
// 提供两种格式的 ID 生成:int64 和 string
// 为什么使用接口:
// - 抽象算法:可以轻松切换不同的 ID 生成算法
// - 便于测试:可以 mock ID 生成器
// - 统一接口:无论底层实现如何,使用方式一致
type IDGenerator interface {
	// NextID 生成一个新的唯一 int64 ID
	// 返回:
	//   int64: 唯一 ID
	//     - 64 位整数
	//     - 按时间递增
	//     - 在当前节点内唯一
	//     - 如果配置了不同的 nodeID,在集群内唯一
	// 使用场景:
	//   - 数据库主键
	//   - 用户 ID
	//   - 订单号
	NextID() int64

	// NextString 生成一个新的唯一 ID,以字符串形式返回
	// 内部调用 NextID() 并转换为字符串
	// 返回:
	//   string: 字符串格式的唯一 ID
	// 使用场景:
	//   - TraceID (请求追踪 ID)
	//   - 需要字符串格式的场景
	//   - JSON 序列化(避免 JavaScript 精度问题)
	// 注意:
	//   JavaScript Number 只能精确表示 53 位整数
	//   如果前端使用,建议使用 string 格式
	NextIDString() string
}

// snowflakeGenerator 的 IDGenerator 基于 Twitter 的 Snowflake 算法实现 Generator
// Snowflake ID 结构(64位):
// - 1 位:未使用(始终为0)
// - 41 位:时间戳(毫秒级,可用约 69 年)
// - 10 位:节点 ID(支持 1024 个节点)
// - 12 位:序列号(每毫秒可生成 4096 个 ID)
// 这种设计保证了:
// - 时间递增:按生成时间排序
// - 分布式唯一:不同节点不会冲突
// - 高性能:每毫秒可生成 4096 个 ID
type snowflakeGenerator struct {
	// node Snowflake 节点实例
	// 每个节点都有唯一的 nodeID
	node *snowflake.Node
}

// NewSnowflake 创建一个新的基于 Snowflake 的 ID 生成器
// 参数:
//
//	nodeID: 节点 ID,必须在分布式系统中唯一
//	  - 取值范围: 0-1023 (10位)
//	  - 单机环境:可以使用 1
//	  - 分布式环境:每个实例使用不同的 ID
//	    例如:从配置文件读取,或根据 IP 自动生成
//
// 返回:
//
//	IDGenerator: ID 生成器接口
//	error: 创建失败时的错误
//	  - nodeID 超出范围(>1023)
//
// 使用示例:
//
//	// 单机环境
//	gen, err := id.NewSnowflake(1)
//
//	// 分布式环境(从配置读取)
//	gen, err := id.NewSnowflake(config.NodeID)
//
//	// K8s 环境(从 Pod 序号)
//	podIndex := getPodIndex() // 0-1023
//	gen, err := id.NewSnowflake(int64(podIndex))
//
// 注意事项:
//   - 相同 nodeID 的实例会生成冲突的 ID
//   - 时钟回拨会导致 ID 重复,生产环境需要注意
//   - 确保系统时间同步(使用 NTP)
func NewSnowflake(nodeID int64) (IDGenerator, error) {
	// 创建 Snowflake 节点
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		// nodeID 无效或超出范围
		return nil, err
	}
	return &snowflakeGenerator{node: node}, nil
}

// NextID 生成一个新的唯一 int64 ID
// 实现 Generator 接口
// 返回:
//
//	int64: Snowflake ID 数值。
//
// 线程安全:
//
//	底层 snowflake.Node 是线程安全的
//	可以在多个 goroutine 中并发调用
func (g *snowflakeGenerator) NextID() int64 {
	// Generate() 生成新的 Snowflake ID
	// Int64() 转换为 int64 类型
	return g.node.Generate().Int64()
}

// NextIDString 生成一个新的唯一 ID,以字符串形式返回
// 实现 Generator 接口
// 返回:
//
//	string: 十进制字符串格式的 ID
//
// 使用场景:
//   - TraceID (中间件中使用)
//   - 前端需要的 ID (避免 JavaScript 精度问题)
func (g *snowflakeGenerator) NextIDString() string {
	// 先生成 int64 ID
	id := g.NextID()
	// 转换为十进制字符串
	// 调用 strconv.FormatInt(id, 10) 转换为十进制字符串。
	// - id: 要转换的整数
	// - 10: 十进制(可以是 2-36,表示不同进制)
	return strconv.FormatInt(id, 10)
}

// DefaultSnowflake 创建一个使用 nodeID 1 的默认生成器
// 这是一个便捷函数,适合单机环境或快速开始
// 返回:
//
//	IDGenerator: 默认的 ID 生成器
//
// 注意:
//
//	如果创建失败会 panic
//	这是因为 nodeID=1 是有效的,不应该失败
//	如果失败说明库有问题,应该立即暴露
//
// 使用场景:
//   - 开发和测试环境
//   - 单机部署
//   - 不需要考虑分布式的场景
//
// 不适用场景:
//   - 多实例部署(会生成重复 ID)
//   - 需要自定义 nodeID 的场景
func DefaultSnowflake() IDGenerator {
	gen, err := NewSnowflake(1)
	if err != nil {
		// 使用 nodeID=1 不应该失败
		// 如果失败了,说明有严重问题,应该 panic
		panic("failed to create default snowflake generator: " + err.Error())
	}
	return gen
}
