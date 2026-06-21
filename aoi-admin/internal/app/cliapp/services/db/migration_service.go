package db

import (
	"context"
	"fmt"
	"io"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	"github.com/rei0721/go-scaffold/pkg/migrator"
)

// RunMigration 执行迁移或输出迁移状态。
func RunMigration(ctx context.Context, configPath string, operation string, stdout io.Writer) error {
	core, err := initapp.NewCore(configPath)
	if err != nil {
		return fmt.Errorf("initialize core: %w", err)
	}
	defer func() {
		if core.Logger != nil {
			_ = core.Logger.Sync()
		}
	}()
	database, err := initapp.NewDatabase(core.Config)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()
	core.Config.Migration.ApplyDefaults()
	runner, err := migrator.New(database, migrator.Config{Driver: string(core.Config.Database.Driver), Dir: core.Config.Migration.Dir})
	if err != nil {
		return err
	}
	switch operation {
	case "up":
		if err := runner.Up(ctx); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "migrations applied")
	case "down":
		if err := runner.Down(ctx); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "migration rolled back")
	case "status":
		return runner.Status(ctx, stdout)
	}
	return nil
}
