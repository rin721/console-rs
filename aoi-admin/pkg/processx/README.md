# pkg/processx - 进程状态辅助

`pkg/processx` 封装少量进程探针能力，当前用于 System Center CLI 判断受管服务是否仍是同一个进程。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`CreateTime`、`IsRunning`。
- 当前风险：[CONFIRMED] 结果依赖操作系统进程表和当前用户权限。
- 非目标：[CONFIRMED] 本包不负责启动、停止、重启进程，也不读取服务日志。

## 使用示例

```go
createTime, err := processx.CreateTime(pid)
if err != nil {
    return err
}

running, err := processx.IsRunning(pid, createTime)
if err != nil {
    return err
}
fmt.Println(running)
```

`IsRunning` 在 `createTime` 大于 0 时会同时比对进程创建时间，降低 PID 复用导致的误判风险。
