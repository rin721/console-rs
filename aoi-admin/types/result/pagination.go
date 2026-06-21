package result

// 本文件定义统一 API 响应结构与 Gin 输出助手，约束状态码、traceId 和分页响应格式。

// Pagination 表示分页元数据
// 包含了前端显示分页控件所需的所有信息
// 设计遵循常见的分页约定
type Pagination struct {
	// Page 当前页码
	// 从 1 开始计数,不是从 0 开始
	// 这符合用户的直觉(第1页、第2页...),而不是程序员思维(索引0、1...)
	Page int `json:"page"`

	// PageSize 每页大小
	// 即每页包含的记录数
	// 常见值: 10, 20, 50, 100
	// 前端可以让用户选择每页显示多少条
	PageSize int `json:"pageSize"`

	// Total 总记录数
	// 使用 int64 而不是 int,支持大数据量
	// 例如:数据库中有 100,000 条用户记录
	// 这个值来自数据库的 COUNT(*) 查询
	Total int64 `json:"total"`

	// TotalPages 总页数
	// 根据 Total 和 PageSize 计算得出
	// 前端可以用这个值显示"共 X 页"
	// 也用于判断是否有下一页(当前页 < 总页数)
	TotalPages int `json:"totalPages"`
}

// PageResult 表示包含列表和分页信息的分页结果
// 使用泛型支持任意类型的列表
// 这是分页查询的标准响应格式
// 类型参数:
//
//	T: 列表项的类型,例如 UserResponse, ProductResponse 等
type PageResult[T any] struct {
	// List 当前页的数据列表
	// 切片类型,包含 0 到 PageSize 个元素
	// 如果是最后一页,可能少于 PageSize 个元素
	// 如果没有数据,是空切片 [] 而不是 nil
	List []T `json:"list"`

	// Pagination 分页元数据
	// 包含页码、总数等信息
	// 前端据此渲染分页控件
	Pagination Pagination `json:"pagination"`
}

// NewPageResult 创建一个新的 PageResult
// 这是一个便捷函数,自动计算总页数
// 参数:
//
//	list: 当前页的数据列表
//	page: 当前页码(从 1 开始)
//	pageSize: 每页大小
//	total: 总记录数(来自数据库 COUNT 查询)
//
// 返回:
//
//	*PageResult[T]: 包含列表和分页信息的完整结果
//
// 使用示例:
//
//	users := []UserResponse{...}  // 从数据库查询的当前页数据
//	total := int64(1000)          // 数据库中的总记录数
//	pageResult := NewPageResult(users, 1, 10, total)
//	return c.JSON(200, result.Success(pageResult))
//
// 响应格式:
//
//	{
//	  "code": 0,
//	  "message": "success",
//	  "data": {
//	    "list": [...],
//	    "pagination": {
//	      "page": 1,
//	      "pageSize": 10,
//	      "total": 1000,
//	      "totalPages": 100
//	    }
//	  },
//	  "serverTime": 1640000000
//	}
func NewPageResult[T any](list []T, page, pageSize int, total int64) *PageResult[T] {
	// 计算总页数
	// 使用整数除法,如果有余数则总页数 +1
	// 例如:
	//   total=100, pageSize=10 => totalPages=10 (刚好整除)
	//   total=105, pageSize=10 => totalPages=11 (有余数,需要多一页)
	//   total=0,   pageSize=10 => totalPages=0  (没有数据)
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		// 如果有余数,说明最后一页不满,但仍需要一页来显示
		// 例如: 105 条数据,每页 10 条,需要 11 页
		// 第 11 页只有 5 条数据
		totalPages++
	}

	// 创建并返回 PageResult
	return &PageResult[T]{
		List: list,
		Pagination: Pagination{
			Page:       page,       // 当前页码
			PageSize:   pageSize,   // 每页大小
			Total:      total,      // 总记录数
			TotalPages: totalPages, // 计算出的总页数
		},
	}
}
