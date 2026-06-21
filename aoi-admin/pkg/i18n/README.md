# pkg/i18n

`pkg/i18n` 是项目的通用国际化基础包，只负责加载资源、解析语言、执行模板插值和记录缺失 key。它不保存业务文案，也不决定产品名称。

## Namespaces

资源按命名空间拆分：

- `ui`：WebUI、CLI、初始化向导、按钮、表单、帮助说明。
- `api`：接口成功、失败、认证、权限和业务异常消息。
- `validation`：参数校验消息。
- `system`：内置菜单、字典、参数、版本和品牌派生标签。

目录示例：

```text
configs/locales/
  ui/zh-CN.yaml
  ui/en-US.yaml
  api/zh-CN.yaml
  api/en-US.yaml
  validation/zh-CN.yaml
  validation/en-US.yaml
  system/zh-CN.yaml
  system/en-US.yaml
```

## Usage

```go
manager, err := i18n.New(&i18n.Config{
    DefaultLocale:  "zh-CN",
    FallbackLocale: "zh-CN",
    Supported:      []string{"zh-CN", "en-US"},
    Resources: map[string]string{
        i18n.NamespaceUI:         "./configs/locales/ui",
        i18n.NamespaceAPI:        "./configs/locales/api",
        i18n.NamespaceValidation: "./configs/locales/validation",
        i18n.NamespaceSystem:     "./configs/locales/system",
    },
})
if err != nil {
    return err
}

message := manager.Localize("zh-CN", i18n.NamespaceAPI, "common.success", nil)
```

缺失 key 会返回 key 本身并记录到 `MissingKeys()`，不会导致服务崩溃。
