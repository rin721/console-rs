package db

import (
	"context"
	"errors"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/database"
)

func TestDatabaseSQLUsesSQLGenDDL(t *testing.T) {
	sql, err := DatabaseSQL(string(database.DriverMySQL), "demo_app")
	if err != nil {
		t.Fatalf("DatabaseSQL() error = %v", err)
	}
	if sql != "CREATE DATABASE IF NOT EXISTS `demo_app`;" {
		t.Fatalf("DatabaseSQL() = %q, want sqlgen CREATE DATABASE", sql)
	}
}

func TestApplyDatabaseRequiresDatabase(t *testing.T) {
	if _, err := ApplyDatabase(context.Background(), nil, string(database.DriverSQLite), "app"); !errors.Is(err, ErrMissingDatabase) {
		t.Fatalf("ApplyDatabase(nil) error = %v, want ErrMissingDatabase", err)
	}
}

func TestDatabaseSQLRejectsUnsupportedDriver(t *testing.T) {
	if _, err := DatabaseSQL("oracle", "app"); !errors.Is(err, ErrUnsupportedDriver) {
		t.Fatalf("DatabaseSQL(unsupported) error = %v, want ErrUnsupportedDriver", err)
	}
}
