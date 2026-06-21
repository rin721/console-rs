package app

// 本文件属于应用组合根，描述 Core、Infrastructure、Modules 与 Transport 在进程内的持有关系。

import "github.com/rei0721/go-scaffold/internal/app/initapp"

// CoreLayer 是核心服务层的公开别名，供 app.App 暴露已装配的核心依赖。
type CoreLayer = initapp.Core

// InfrastructureLayer 是基础设施层的公开别名，包含数据库、缓存、执行器和存储等组件。
type InfrastructureLayer = initapp.Infrastructure

// ModulesLayer 是业务模块层的公开别名。
type ModulesLayer = initapp.Modules

type IAMModule = initapp.IAMModule

type PluginsModule = initapp.PluginsModule

// TransportLayer 是传输层的公开别名，包含 Gin 路由和 HTTP server 包装器。
type TransportLayer = initapp.Transport
