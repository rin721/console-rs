package commands

import (
	"github.com/rei0721/go-scaffold/internal/app/cliapp/localization"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

// NewTopLevelCommands 返回所有顶层 CLI 命令。
func NewTopLevelCommands(localizers ...*localization.Localizer) []cli.CommandSpec {
	localizer := commandLocalizer(localizers...)
	serverSpec := NewServerCommand().Spec()
	serverSpec.HomeHidden = true
	dbSpec := NewDBCommand().Spec()
	dbSpec.HomeHidden = true
	iamSpec := NewIAMCommand().Spec()
	iamSpec.HomeHidden = true
	apiSpec := NewAPICommand()

	specs := []cli.CommandSpec{
		serverSpec,
		dbSpec,
		iamSpec,
		apiSpec,
	}
	specs = append(specs, NewSystemCenterCommands(localizer)...)
	return specs
}

func commandLocalizer(localizers ...*localization.Localizer) *localization.Localizer {
	if len(localizers) > 0 && localizers[0] != nil {
		return localizers[0]
	}
	return localization.ForArgs(nil)
}
