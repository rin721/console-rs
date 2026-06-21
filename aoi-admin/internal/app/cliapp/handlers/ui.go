package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/rei0721/go-scaffold/pkg/cli"
)

func requireUI(ctx *cli.Context) (cli.PromptUI, error) {
	if ctx == nil || ctx.UI == nil {
		return nil, fmt.Errorf("interactive UI is not available")
	}
	if ctx.Context == nil {
		ctx.Context = context.Background()
	}
	return ctx.UI, nil
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultInt(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
