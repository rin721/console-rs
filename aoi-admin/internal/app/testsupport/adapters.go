package testsupport

import (
	"github.com/rei0721/go-scaffold/internal/app/adapters"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/pkg/authorization"
	"github.com/rei0721/go-scaffold/pkg/database"
	"github.com/rei0721/go-scaffold/pkg/token"
	"github.com/rei0721/go-scaffold/pkg/web"
)

// Database 把测试用真实数据库适配为内部端口。
func Database(db database.Database) ports.Database {
	return adapters.NewDatabase(db)
}

// Executor 把测试用数据库执行器适配为内部端口。
func Executor(executor database.Executor) ports.Executor {
	return adapters.NewExecutor(executor)
}

// TokenManager 把测试用 token 管理器适配为 IAM 服务接口。
func TokenManager(manager token.Manager) iamservice.TokenManager {
	return adapters.NewTokenManager(manager)
}

// AuthorizerEnforcer 把测试用授权执行器适配为 IAM 服务接口。
func AuthorizerEnforcer(enforcer authorization.Enforcer) iamservice.AuthorizerEnforcer {
	return adapters.NewAuthorizerEnforcer(enforcer)
}

// TOTPProvider 返回测试可复用的 IAM TOTP 实现。
func TOTPProvider() iamservice.TOTPProvider {
	return adapters.TOTPProvider{}
}

// HTTPRouter 创建测试用 HTTP 引擎和端口化路由。
func HTTPRouter(mode string) (*web.Engine, *adapters.HTTPEngine) {
	engine := web.New(mode)
	return engine, adapters.NewHTTPEngine(engine)
}
