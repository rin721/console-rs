package commands

import (
	"os"
	"path/filepath"

	httptransport "github.com/rei0721/go-scaffold/internal/transport/http"
	"github.com/rei0721/go-scaffold/pkg/cli"
)

func NewAPICommand() cli.CommandSpec {
	return cli.CommandSpec{
		Name:        "api",
		Use:         "api",
		Description: "Generate and inspect API contracts",
		HomeHidden:  true,
		Commands: []cli.CommandSpec{
			newOpenAPICommand(),
		},
	}
}

func newOpenAPICommand() cli.CommandSpec {
	return cli.CommandSpec{
		Name:        "openapi",
		Use:         "openapi [--output=<path>]",
		Description: "Generate the main system OpenAPI contract",
		Example:     "aoi api openapi --output docs/api/openapi.yaml",
		Flags: []cli.FlagSpec{
			{Name: "output", Type: cli.FlagTypeString, Default: "", Description: "Output file path; stdout when empty"},
		},
		Run: runOpenAPICommand,
	}
}

func runOpenAPICommand(ctx *cli.Context) error {
	raw, err := httptransport.GenerateOpenAPIYAML()
	if err != nil {
		return err
	}
	output := ctx.GetString("output")
	if output == "" {
		_, err = ctx.Stdout.Write(raw)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(output, raw, 0o644)
}
