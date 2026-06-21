package testsupport

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/pkg/authorization"
	"github.com/rei0721/go-scaffold/pkg/crypto"
	"github.com/rei0721/go-scaffold/pkg/database"
	"github.com/rei0721/go-scaffold/pkg/mfa"
	"github.com/rei0721/go-scaffold/pkg/migrator"
	"github.com/rei0721/go-scaffold/pkg/token"
	"github.com/rei0721/go-scaffold/pkg/utils"
)

// SQLiteDatabase 创建测试用 SQLite 数据库，并在测试结束时关闭连接。
func SQLiteDatabase(t testing.TB, filename string) database.Database {
	t.Helper()
	if filename == "" {
		filename = "test.db"
	}
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(t.TempDir(), filename)
	}
	db, err := database.New(&database.Config{
		Driver: database.DriverSQLite,
		DBName: filename,
	})
	if err != nil {
		t.Fatalf("create sqlite database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite database: %v", err)
		}
	})
	return db
}

// IAMSQLiteDatabase 创建已执行 IAM 迁移的内部数据库端口。
func IAMSQLiteDatabase(t testing.TB, filename string) ports.Database {
	t.Helper()
	db := SQLiteDatabase(t, filename)
	runner, err := migrator.New(db, migrator.Config{
		Driver: string(database.DriverSQLite),
		Dir:    migrationsDir(t),
	})
	if err != nil {
		t.Fatalf("create migration runner: %v", err)
	}
	if err := runner.Up(context.Background()); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	return Database(db)
}

// IAMDeps 聚合 IAM service 测试所需的最小接口实现。
type IAMDeps struct {
	Passwords iamservice.PasswordCrypto
	Tokens    iamservice.TokenManager
	Authz     iamservice.AuthorizerEnforcer
	IDs       iamservice.IDGenerator
	TOTP      iamservice.TOTPProvider
}

// NewIAMDeps 创建 IAM service 测试默认依赖。
func NewIAMDeps(t testing.TB) IAMDeps {
	t.Helper()
	passwords, err := crypto.NewBcrypt()
	if err != nil {
		t.Fatalf("create password crypto: %v", err)
	}
	tokens, err := token.New(token.Config{
		Issuer:        "test",
		Audience:      []string{"test"},
		SigningKey:    "01234567890123456789012345678901",
		AccessTTL:     time.Hour,
		RefreshTTL:    time.Hour,
		RefreshPepper: "refresh-pepper",
	})
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	authz, err := authorization.New()
	if err != nil {
		t.Fatalf("create authorizer: %v", err)
	}
	ids, err := utils.NewSnowflake(7)
	if err != nil {
		t.Fatalf("create id generator: %v", err)
	}
	return IAMDeps{
		Passwords: passwords,
		Tokens:    TokenManager(tokens),
		Authz:     AuthorizerEnforcer(authz),
		IDs:       ids,
		TOTP:      TOTPProvider(),
	}
}

// IAMTOTPCode 生成测试断言使用的 TOTP code。
func IAMTOTPCode(t testing.TB, secret string, at time.Time) string {
	t.Helper()
	code, err := mfa.GenerateTOTPCode(secret, at)
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}
	return code
}

func migrationsDir(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve testsupport path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}
