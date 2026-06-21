package commands

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/rei0721/go-scaffold/internal/app/cliapp/handlers"
	servicedb "github.com/rei0721/go-scaffold/internal/app/cliapp/services/db"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/types/constants"
)

func TestDBCommandMetadata(t *testing.T) {
	cmd := NewDBCommand()

	if got := cmd.Name(); got != dbCommandName {
		t.Fatalf("Name() = %q, want %q", got, dbCommandName)
	}
	if !strings.Contains(cmd.Description(), "sqlgen-powered") {
		t.Fatalf("Description() = %q, want sqlgen wording", cmd.Description())
	}
	if !strings.Contains(cmd.Usage(), "--operation") {
		t.Fatalf("Usage() = %q, want operation flag", cmd.Usage())
	}
	if !strings.Contains(cmd.Usage(), servicedb.OperationDatabase) {
		t.Fatalf("Usage() = %q, want database operation", cmd.Usage())
	}

	gotFlags := cmd.Flags()
	wantNames := []string{"config", "operation", "apply", "print-sql"}
	if len(gotFlags) != len(wantNames) {
		t.Fatalf("len(Flags()) = %d, want %d", len(gotFlags), len(wantNames))
	}
	for i, want := range wantNames {
		if gotFlags[i].Name != want {
			t.Fatalf("Flags()[%d].Name = %q, want %q", i, gotFlags[i].Name, want)
		}
	}
	if gotFlags[0].Default != constants.AppDefaultConfigPath {
		t.Fatalf("config default = %v, want %q", gotFlags[0].Default, constants.AppDefaultConfigPath)
	}
	if gotFlags[1].Default != servicedb.DefaultOperation {
		t.Fatalf("operation default = %v, want %q", gotFlags[1].Default, servicedb.DefaultOperation)
	}
}

func TestDBSQLForPrintDatabaseOperation(t *testing.T) {
	sql, err := servicedb.SQLForPrint(servicedb.OperationOptions{Operation: servicedb.OperationDatabase}, appconfig.DatabaseConfig{
		Driver: "mysql",
		MySQL: appconfig.DatabaseMySQLConfig{
			Database: "demo_app",
		},
	})
	if err != nil {
		t.Fatalf("dbSQLForPrint() error = %v", err)
	}
	if sql != "CREATE DATABASE IF NOT EXISTS `demo_app`;" {
		t.Fatalf("dbSQLForPrint() = %q, want sqlgen CREATE DATABASE", sql)
	}
}

func TestDBCommandExecutePassesOptionsToRunner(t *testing.T) {
	var got servicedb.OperationOptions
	cmd := &DBCommand{Handler: &handlers.DBHandler{
		Runner: func(_ context.Context, opts servicedb.OperationOptions) (servicedb.OperationResult, error) {
			got = opts
			return servicedb.OperationResult{Message: "ok"}, nil
		},
	}}

	var stdout bytes.Buffer
	err := cmd.Handler.Execute(&cli.Context{
		Flags: map[string]interface{}{
			"config":    "configs/test.yaml",
			"operation": servicedb.OperationDatabase,
			"apply":     true,
			"print-sql": true,
		},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := servicedb.OperationOptions{
		ConfigPath: "configs/test.yaml",
		Operation:  servicedb.OperationDatabase,
		Apply:      true,
		PrintSQL:   true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("options = %#v, want %#v", got, want)
	}
}

func TestDBCommandExecuteDefaultsOperation(t *testing.T) {
	var got servicedb.OperationOptions
	cmd := &DBCommand{Handler: &handlers.DBHandler{
		Runner: func(_ context.Context, opts servicedb.OperationOptions) (servicedb.OperationResult, error) {
			got = opts
			return servicedb.OperationResult{}, nil
		},
	}}

	err := cmd.Handler.Execute(&cli.Context{
		Flags: map[string]interface{}{
			"config":    constants.AppDefaultConfigPath,
			"operation": "",
			"apply":     false,
			"print-sql": false,
		},
		Stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Operation != servicedb.DefaultOperation {
		t.Fatalf("operation = %q, want %q", got.Operation, servicedb.DefaultOperation)
	}
}
