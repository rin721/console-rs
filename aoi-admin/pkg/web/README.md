# pkg/web - HTTP 防腐封装

`pkg/web` 是项目 HTTP 传输层的防腐边界。它把 Gin 和 gin-contrib/cors 收敛在 `pkg/` 内部,向 `internal/` 暴露项目自有的 `web.Context`、`web.HandlerFunc`、`web.Router` 和 `web.Engine`。

## 边界原则

- `internal/` 代码注册路由、读取请求、返回 JSON 时只依赖 `pkg/web`。
- 中间件返回 `web.HandlerFunc`,不把底层 Gin handler 作为公共契约。
- CORS 配置使用 `web.CORSConfig`,避免 gin-contrib/cors 的配置类型泄漏到应用层。
- `web.Engine` 实现 `http.Handler`,可以直接交给标准库 `http.Server`。

## 基本使用

```go
router := web.New("release")
router.Use(web.Recovery())
router.Use(web.CORS(web.CORSConfig{
    Enabled:      true,
    AllowOrigins: []string{"https://example.com"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
    MaxAge:       12 * 60 * 60,
}))

router.GET("/health", func(c web.Context) {
    c.JSON(http.StatusOK, map[string]any{"status": "healthy"})
})

server := &http.Server{
    Addr:    ":8080",
    Handler: router,
}
```

## Handler 约定

```go
func createTodo(service *TodoService) web.HandlerFunc {
    return func(c web.Context) {
        var req CreateTodoRequest
        if err := c.BindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
            return
        }

        todo, err := service.Create(c.RequestContext(), req)
        if err != nil {
            c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
            return
        }

        c.JSON(http.StatusCreated, todo)
    }
}
```

## 扩展规则

如果传输层需要新的能力,优先在 `web.Context` 或 `web.Router` 中增加项目自有方法,再在适配器里映射到底层框架。不要从 `pkg/web` 返回底层框架对象给 `internal/` 使用。
