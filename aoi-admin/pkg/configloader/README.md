# pkg/configloader - 配置加载防腐封装

`pkg/configloader` 是配置文件、dotenv 和文件监听能力的防腐边界。它把 Viper、fsnotify 和 godotenv 收敛在 `pkg/` 内部,向 `internal/config` 暴露项目自有的 `Loader` 和 `Event`。

## 边界原则

- `internal/` 只依赖 `configloader.Loader`,不持有底层配置解析器对象。
- 文件变更事件统一转换为 `configloader.Event`,不向上层泄漏 fsnotify 类型。
- `.env` 加载通过 `configloader.LoadEnv(path)` 触发,调用方不直接依赖 godotenv。
- 如果需要新增配置能力,优先扩展本包方法,不要让第三方配置库类型进入 `internal/`。

## 基本使用

```go
configloader.LoadEnv(".env")

loader := configloader.New()
loader.SetConfigFile("configs/config.yaml")

if err := loader.ReadInConfig(); err != nil {
    return err
}

loader.Set("server.host", "127.0.0.1")

var cfg AppConfig
if err := loader.Unmarshal(&cfg); err != nil {
    return err
}
```

## 配置监听

```go
loader.OnConfigChange(func(event configloader.Event) {
    log.Printf("config changed: %s %s", event.Name, event.Op)
})
loader.WatchConfig()
```

`Event.Op` 是字符串形式的操作描述,用于日志和重载判断。调用方不需要知道底层监听库的枚举类型。
