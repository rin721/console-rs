// Package authorization wraps Casbin behind the project's RBAC interface.
package authorization

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

const casbinModel = `
[request_definition]
r = sub, dom, product, scope, obj, act

[policy_definition]
p = sub, dom, product, scope, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && p.product == r.product && p.scope == r.scope && (p.obj == "*" || keyMatch2(r.obj, p.obj)) && (p.act == "*" || regexMatch(r.act, p.act))
`

// Rule represents one Casbin policy or grouping rule loaded from storage.
type Rule struct {
	PType  string
	Values []string
}

// Enforcer defines organization-domain RBAC with product and permission scope.
type Enforcer interface {
	Enforce(ctx context.Context, sub, org, product, scope, obj, act string) (bool, error)
	AddPolicy(ctx context.Context, role, org, product, scope, obj, act string) (bool, error)
	AddRoleForUser(ctx context.Context, user, role, org string) (bool, error)
	DeleteRoleForUser(ctx context.Context, user, role, org string) (bool, error)
	GetRolesForUser(ctx context.Context, user, org string) ([]string, error)
	LoadRules(ctx context.Context, rules []Rule) error
}

type enforcer struct {
	inner *casbin.SyncedEnforcer
}

// New creates the in-memory domain RBAC enforcer. Policies are loaded by callers.
func New() (Enforcer, error) {
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		return nil, fmt.Errorf("build casbin model: %w", err)
	}
	e, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("build casbin enforcer: %w", err)
	}
	return &enforcer{inner: e}, nil
}

func (e *enforcer) Enforce(_ context.Context, sub, org, product, scope, obj, act string) (bool, error) {
	return e.inner.Enforce(sub, org, product, scope, obj, act)
}

func (e *enforcer) AddPolicy(_ context.Context, role, org, product, scope, obj, act string) (bool, error) {
	return e.inner.AddPolicy(role, org, product, scope, obj, act)
}

func (e *enforcer) AddRoleForUser(_ context.Context, user, role, org string) (bool, error) {
	return e.inner.AddRoleForUserInDomain(user, role, org)
}

func (e *enforcer) DeleteRoleForUser(_ context.Context, user, role, org string) (bool, error) {
	return e.inner.DeleteRoleForUserInDomain(user, role, org)
}

func (e *enforcer) GetRolesForUser(_ context.Context, user, org string) ([]string, error) {
	return e.inner.GetRolesForUserInDomain(user, org), nil
}

// LoadRules accepts only p and g rules and rebuilds the in-memory policy state.
func (e *enforcer) LoadRules(_ context.Context, rules []Rule) error {
	e.inner.ClearPolicy()
	for _, rule := range rules {
		switch rule.PType {
		case "p":
			if len(rule.Values) < 6 {
				return fmt.Errorf("invalid p rule: %#v", rule.Values)
			}
			if _, err := e.inner.AddPolicy(rule.Values[0], rule.Values[1], rule.Values[2], rule.Values[3], rule.Values[4], rule.Values[5]); err != nil {
				return err
			}
		case "g":
			if len(rule.Values) < 3 {
				return fmt.Errorf("invalid g rule: %#v", rule.Values)
			}
			if _, err := e.inner.AddRoleForUserInDomain(rule.Values[0], rule.Values[1], rule.Values[2]); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported casbin rule type: %s", rule.PType)
		}
	}
	return nil
}
