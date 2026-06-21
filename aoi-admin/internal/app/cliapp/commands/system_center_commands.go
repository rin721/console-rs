package commands

import (
	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

// NewSystemCenterCommands 返回系统中心相关的交互式运维命令集合。
func NewSystemCenterCommands(localizers ...*localization.Localizer) []cli.CommandSpec {
	localizer := commandLocalizer(localizers...)
	return []cli.CommandSpec{
		newRunCommandSpec(localizer),
		newServiceCommandSpec(localizer),
		newInitCommandSpec(localizer),
	}
}
