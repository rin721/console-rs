package initapp

import (
	"context"
	"fmt"

	"github.com/rei0721/go-scaffold/pkg/migrator"
)

func ApplyExplicitMigrations(ctx context.Context, core Core, infra Infrastructure) error {
	return applyMigrations(ctx, core, infra, "initial-setup")
}

func applyMigrations(ctx context.Context, core Core, infra Infrastructure, trigger string) error {
	core.Config.Migration.ApplyDefaults()
	runner, err := migrator.New(infra.Database, migrator.Config{
		Driver: string(core.Config.Database.Driver),
		Dir:    core.Config.Migration.Dir,
	})
	if err != nil {
		return fmt.Errorf("initialize migrator: %w", err)
	}
	if err := runner.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if core.Logger != nil {
		core.Logger.Info("database migrations applied", "dir", core.Config.Migration.Dir, "trigger", trigger)
	}
	return nil
}
