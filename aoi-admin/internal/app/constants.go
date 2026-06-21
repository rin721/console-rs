package app

// 本文件属于应用组合根，描述 Core、Infrastructure、Modules 与 Transport 在进程内的持有关系。

import "github.com/rei0721/go-scaffold/internal/app/mainapp"

// AppMode 暴露应用运行模式类型，同时隐藏内部 mainapp 子包路径。
type AppMode = mainapp.Mode

const (
	// ModeServer 是当前脚手架唯一对外稳定的服务运行模式。
	ModeServer = mainapp.ModeServer
)

const (
	// ConstantsI18nResourcesDir 是默认本地化资源根目录。
	ConstantsI18nResourcesDir = "./configs/locales"
	// ConstantsI18nDefaultLanguage 是默认语言标识。
	ConstantsI18nDefaultLanguage = "zh-CN"
	// ConstantsDefaultHost 是本地启动时的默认监听主机。
	ConstantsDefaultHost = "localhost"
)

// ConstantsI18nSupportedLanguages 是脚手架默认随配置暴露的语言集合。
var ConstantsI18nSupportedLanguages = []string{"zh-CN", "en-US"}
