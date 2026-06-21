package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/iam/model"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	database "github.com/rei0721/go-scaffold/internal/ports"
)

type Repository = iamservice.Repository

type AuditLogFilter = iamservice.AuditLogFilter

type APITokenFilter = iamservice.APITokenFilter

type repository struct {
	db database.Executor
}

func New(db database.Executor) Repository {
	return &repository{db: iamExecutor{inner: db}}
}

func (r *repository) WithExecutor(db database.Executor) Repository {
	return &repository{db: iamExecutor{inner: db}}
}

func (r *repository) WithTx(ctx context.Context, fn func(context.Context, iamservice.Repository) error) error {
	if fn == nil {
		return database.ErrNilTxFunc
	}
	if txer, ok := r.db.(interface {
		WithTx(context.Context, database.TxFunc) error
	}); ok {
		return txer.WithTx(ctx, func(txCtx context.Context, tx database.Executor) error {
			return fn(txCtx, r.WithExecutor(tx))
		})
	}
	return fn(ctx, r)
}

type iamExecutor struct {
	inner database.Executor
}

func (e iamExecutor) Create(ctx context.Context, value any) error {
	return mapRepositoryError(e.inner.Create(ctx, value))
}

func (e iamExecutor) Save(ctx context.Context, value any) error {
	return mapRepositoryError(e.inner.Save(ctx, value))
}

func (e iamExecutor) First(ctx context.Context, dest any, opts ...database.QueryOption) error {
	return mapRepositoryError(e.inner.First(ctx, dest, opts...))
}

func (e iamExecutor) Find(ctx context.Context, dest any, opts ...database.QueryOption) error {
	return mapRepositoryError(e.inner.Find(ctx, dest, opts...))
}

func (e iamExecutor) Update(ctx context.Context, model any, values map[string]any, opts ...database.QueryOption) (database.Result, error) {
	result, err := e.inner.Update(ctx, model, values, opts...)
	return result, mapRepositoryError(err)
}

func (e iamExecutor) Delete(ctx context.Context, model any, opts ...database.QueryOption) (database.Result, error) {
	result, err := e.inner.Delete(ctx, model, opts...)
	return result, mapRepositoryError(err)
}

func (e iamExecutor) Exec(ctx context.Context, sql string, args ...any) (database.Result, error) {
	result, err := e.inner.Exec(ctx, sql, args...)
	return result, mapRepositoryError(err)
}

func (e iamExecutor) Raw(ctx context.Context, dest any, sql string, args ...any) (database.Result, error) {
	result, err := e.inner.Raw(ctx, dest, sql, args...)
	return result, mapRepositoryError(err)
}

func (e iamExecutor) Count(ctx context.Context, model any, opts ...database.QueryOption) (int64, error) {
	count, err := e.inner.Count(ctx, model, opts...)
	return count, mapRepositoryError(err)
}

func (e iamExecutor) HasTable(ctx context.Context, model any) (bool, error) {
	ok, err := e.inner.HasTable(ctx, model)
	return ok, mapRepositoryError(err)
}

func (e iamExecutor) WithTx(ctx context.Context, fn database.TxFunc) error {
	if fn == nil {
		return database.ErrNilTxFunc
	}
	txer, ok := e.inner.(interface {
		WithTx(context.Context, database.TxFunc) error
	})
	if !ok {
		return fn(ctx, e)
	}
	return txer.WithTx(ctx, func(txCtx context.Context, tx database.Executor) error {
		return fn(txCtx, iamExecutor{inner: tx})
	})
}

func mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, database.ErrNotFound) {
		return iamservice.ErrNotFound
	}
	return err
}

func (r *repository) HasUsersTable(ctx context.Context) (bool, error) {
	return r.db.HasTable(ctx, &model.User{})
}

func (r *repository) CreateOrganization(ctx context.Context, org *model.Organization) error {
	return r.db.Create(ctx, org)
}

func (r *repository) FindOrganizationByID(ctx context.Context, id int64) (*model.Organization, error) {
	var org model.Organization
	if err := r.db.First(ctx, &org, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *repository) FindOrganizationByCode(ctx context.Context, code string) (*model.Organization, error) {
	var org model.Organization
	if err := r.db.First(ctx, &org, database.Where("code = ?", code), alive()); err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *repository) ListOrganizations(ctx context.Context) ([]model.Organization, error) {
	var orgs []model.Organization
	err := r.db.Find(ctx, &orgs, alive(), database.Order("id DESC"))
	return orgs, err
}

func (r *repository) SaveOrganization(ctx context.Context, org *model.Organization) error {
	org.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, org)
}

func (r *repository) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.Create(ctx, user)
}

func (r *repository) CountUsers(ctx context.Context) (int64, error) {
	return r.db.Count(ctx, &model.User{}, alive())
}

func (r *repository) FindUserByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	if err := r.db.First(ctx, &user, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) FindUserByIdentifier(ctx context.Context, identifier string) (*model.User, error) {
	var user model.User
	if err := r.db.First(ctx, &user, database.Where("(username = ? OR email = ?)", identifier, identifier), alive()); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) SaveUser(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, user)
}

func (r *repository) CreateMembership(ctx context.Context, membership *model.Membership) error {
	return r.db.Create(ctx, membership)
}

func (r *repository) FindMembership(ctx context.Context, orgID, userID int64) (*model.Membership, error) {
	var membership model.Membership
	err := r.db.First(ctx, &membership,
		database.Where("org_id = ? AND user_id = ?", orgID, userID),
		database.Where("status = ?", model.StatusActive),
		alive(),
	)
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func (r *repository) FindMembershipAnyStatus(ctx context.Context, orgID, userID int64) (*model.Membership, error) {
	var membership model.Membership
	err := r.db.First(ctx, &membership,
		database.Where("org_id = ? AND user_id = ?", orgID, userID),
		alive(),
	)
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func (r *repository) ListMembershipsByUser(ctx context.Context, userID int64) ([]model.Membership, error) {
	var memberships []model.Membership
	err := r.db.Find(ctx, &memberships,
		database.Where("user_id = ?", userID),
		database.Where("status = ?", model.StatusActive),
		alive(),
	)
	return memberships, err
}

func (r *repository) ListMembershipsByOrg(ctx context.Context, orgID int64) ([]model.Membership, error) {
	var memberships []model.Membership
	err := r.db.Find(ctx, &memberships,
		database.Where("org_id = ?", orgID),
		alive(),
		database.Order("id DESC"),
	)
	return memberships, err
}

func (r *repository) SaveMembership(ctx context.Context, membership *model.Membership) error {
	membership.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, membership)
}

func (r *repository) ListUsersByOrg(ctx context.Context, orgID int64) ([]model.User, error) {
	var users []model.User
	_, err := r.db.Raw(ctx, &users, `
SELECT u.*
FROM iam_users u
JOIN iam_memberships m ON m.user_id = u.id
WHERE m.org_id = ? AND m.status = ? AND m.deleted_at IS NULL AND u.deleted_at IS NULL
ORDER BY u.id DESC`, orgID, model.StatusActive)
	return users, err
}

func (r *repository) CreateRole(ctx context.Context, role *model.Role) error {
	return r.db.Create(ctx, role)
}

func (r *repository) FindRoleByID(ctx context.Context, id int64) (*model.Role, error) {
	var role model.Role
	if err := r.db.First(ctx, &role, database.Where("id = ?", id), alive()); err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *repository) FindRole(ctx context.Context, orgID int64, code string) (*model.Role, error) {
	var role model.Role
	if err := r.db.First(ctx, &role, database.Where("org_id = ? AND code = ?", orgID, code), alive()); err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *repository) ListRoles(ctx context.Context, orgID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Find(ctx, &roles, database.Where("org_id = ?", orgID), alive(), database.Order("code ASC"))
	return roles, err
}

func (r *repository) ListRolePermissions(ctx context.Context, orgID int64, roleSubject string) ([]iamservice.RolePermission, error) {
	var rows []model.CasbinRule
	if err := r.db.Find(ctx, &rows,
		database.Where("ptype = ? AND v0 = ? AND v1 = ?", "p", roleSubject, strconv.FormatInt(orgID, 10)),
		database.Order("v2 ASC, v3 ASC, v4 ASC, v5 ASC"),
	); err != nil {
		return nil, err
	}
	permissions := make([]iamservice.RolePermission, 0, len(rows))
	for _, row := range rows {
		if row.V4 == "" || row.V5 == "" {
			continue
		}
		permissions = append(permissions, iamservice.RolePermission{
			ProductCode: row.V2,
			Scope:       row.V3,
			Code:        row.V4 + ":" + row.V5,
		})
	}
	return permissions, nil
}

func (r *repository) SaveRole(ctx context.Context, role *model.Role) error {
	role.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, role)
}

func (r *repository) CreatePermission(ctx context.Context, permission *model.Permission) error {
	return r.db.Create(ctx, permission)
}

func (r *repository) FindPermission(ctx context.Context, productCode, scope, code string) (*model.Permission, error) {
	var permission model.Permission
	if err := r.db.First(ctx, &permission, database.Where("product_code = ? AND scope = ? AND code = ?", productCode, scope, code)); err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *repository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.Find(ctx, &permissions, database.Order("product_code ASC, scope ASC, code ASC"))
	return permissions, err
}

func (r *repository) CreateSession(ctx context.Context, session *model.Session) error {
	return r.db.Create(ctx, session)
}

func (r *repository) FindSessionByID(ctx context.Context, id int64) (*model.Session, error) {
	var session model.Session
	if err := r.db.First(ctx, &session, database.Where("id = ?", id)); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *repository) FindSessionByRefreshHash(ctx context.Context, hash string) (*model.Session, error) {
	var session model.Session
	if err := r.db.First(ctx, &session, database.Where("refresh_token_hash = ?", hash)); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *repository) ListSessionsByOrg(ctx context.Context, orgID int64) ([]model.Session, error) {
	var sessions []model.Session
	err := r.db.Find(ctx, &sessions, database.Where("org_id = ?", orgID), database.Order("created_at DESC"))
	return sessions, err
}

func (r *repository) ListSessionsByUser(ctx context.Context, userID int64) ([]model.Session, error) {
	var sessions []model.Session
	err := r.db.Find(ctx, &sessions, database.Where("user_id = ?", userID), database.Order("created_at DESC"))
	return sessions, err
}

func (r *repository) SaveSession(ctx context.Context, session *model.Session) error {
	session.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, session)
}

func (r *repository) CreateAPIToken(ctx context.Context, apiToken *model.APIToken) error {
	return r.db.Create(ctx, apiToken)
}

func (r *repository) FindAPITokenByHash(ctx context.Context, hash string) (*model.APIToken, error) {
	var apiToken model.APIToken
	if err := r.db.First(ctx, &apiToken, database.Where("token_hash = ?", hash)); err != nil {
		return nil, err
	}
	return &apiToken, nil
}

func (r *repository) FindAPITokenByID(ctx context.Context, id int64) (*model.APIToken, error) {
	var apiToken model.APIToken
	if err := r.db.First(ctx, &apiToken, database.Where("id = ?", id)); err != nil {
		return nil, err
	}
	return &apiToken, nil
}

func (r *repository) ListAPITokens(ctx context.Context, orgID int64, filter APITokenFilter) ([]model.APIToken, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 10
	}
	if filter.Now.IsZero() {
		filter.Now = time.Now().UTC()
	}
	opts := []database.QueryOption{
		database.Where("org_id = ?", orgID),
	}
	if filter.UserID > 0 {
		opts = append(opts, database.Where("user_id = ?", filter.UserID))
	}
	switch filter.Status {
	case model.StatusActive:
		opts = append(opts,
			database.Where("status = ?", model.StatusActive),
			database.Where("(expires_at IS NULL OR expires_at >= ?)", filter.Now),
		)
	case model.StatusExpired:
		opts = append(opts,
			database.Where("status = ?", model.StatusActive),
			database.Where("expires_at IS NOT NULL AND expires_at < ?", filter.Now),
		)
	case model.StatusRevoked:
		opts = append(opts, database.Where("status = ?", model.StatusRevoked))
	}
	total, err := r.db.Count(ctx, &model.APIToken{}, opts...)
	if err != nil {
		return nil, 0, err
	}
	query := append(append([]database.QueryOption{}, opts...),
		database.Order("created_at DESC, id DESC"),
		database.Limit(filter.PageSize),
		database.Offset((filter.Page-1)*filter.PageSize),
	)
	var items []model.APIToken
	if err := r.db.Find(ctx, &items, query...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *repository) SaveAPIToken(ctx context.Context, apiToken *model.APIToken) error {
	apiToken.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, apiToken)
}

func (r *repository) CreateInvitation(ctx context.Context, invitation *model.Invitation) error {
	return r.db.Create(ctx, invitation)
}

func (r *repository) FindInvitationByID(ctx context.Context, id int64) (*model.Invitation, error) {
	var invitation model.Invitation
	if err := r.db.First(ctx, &invitation, database.Where("id = ?", id)); err != nil {
		return nil, err
	}
	return &invitation, nil
}

func (r *repository) FindInvitationByTokenHash(ctx context.Context, hash string) (*model.Invitation, error) {
	var invitation model.Invitation
	if err := r.db.First(ctx, &invitation, database.Where("token_hash = ?", hash)); err != nil {
		return nil, err
	}
	return &invitation, nil
}

func (r *repository) ListInvitationsByOrg(ctx context.Context, orgID int64) ([]model.Invitation, error) {
	var invitations []model.Invitation
	err := r.db.Find(ctx, &invitations, database.Where("org_id = ?", orgID), database.Order("created_at DESC"))
	return invitations, err
}

func (r *repository) SaveInvitation(ctx context.Context, invitation *model.Invitation) error {
	invitation.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, invitation)
}

func (r *repository) CreatePasswordReset(ctx context.Context, reset *model.PasswordReset) error {
	return r.db.Create(ctx, reset)
}

func (r *repository) FindPasswordResetByTokenHash(ctx context.Context, hash string) (*model.PasswordReset, error) {
	var reset model.PasswordReset
	if err := r.db.First(ctx, &reset, database.Where("token_hash = ?", hash)); err != nil {
		return nil, err
	}
	return &reset, nil
}

func (r *repository) SavePasswordReset(ctx context.Context, reset *model.PasswordReset) error {
	reset.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, reset)
}

func (r *repository) CreateEmailVerification(ctx context.Context, verification *model.EmailVerification) error {
	return r.db.Create(ctx, verification)
}

func (r *repository) FindEmailVerificationByTokenHash(ctx context.Context, hash string) (*model.EmailVerification, error) {
	var verification model.EmailVerification
	if err := r.db.First(ctx, &verification, database.Where("token_hash = ?", hash)); err != nil {
		return nil, err
	}
	return &verification, nil
}

func (r *repository) SaveEmailVerification(ctx context.Context, verification *model.EmailVerification) error {
	verification.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, verification)
}

func (r *repository) DeletePendingSignup(ctx context.Context, orgID, userID, verificationID int64) error {
	statements := []struct {
		query string
		args  []any
	}{
		{query: "DELETE FROM iam_email_verifications WHERE id = ? AND user_id = ? AND org_id = ? AND status = ?", args: []any{verificationID, userID, orgID, model.StatusPending}},
		{query: "DELETE FROM iam_memberships WHERE org_id = ? AND user_id = ? AND status = ?", args: []any{orgID, userID, model.StatusPending}},
		{query: "DELETE FROM iam_users WHERE id = ? AND status = ?", args: []any{userID, model.StatusPending}},
		{query: "DELETE FROM iam_organizations WHERE id = ? AND status = ?", args: []any{orgID, model.StatusPending}},
	}
	for _, statement := range statements {
		if _, err := r.db.Exec(ctx, statement.query, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

func (r *repository) CreateMFAFactor(ctx context.Context, factor *model.MFAFactor) error {
	return r.db.Create(ctx, factor)
}

func (r *repository) FindActiveMFAFactor(ctx context.Context, userID int64) (*model.MFAFactor, error) {
	var factor model.MFAFactor
	err := r.db.First(ctx, &factor,
		database.Where("user_id = ?", userID),
		database.Where("type = ?", "totp"),
		database.Where("status = ?", model.StatusActive),
		database.Order("id DESC"),
	)
	if err != nil {
		return nil, err
	}
	return &factor, nil
}

func (r *repository) SaveMFAFactor(ctx context.Context, factor *model.MFAFactor) error {
	factor.UpdatedAt = time.Now().UTC()
	return r.db.Save(ctx, factor)
}

func (r *repository) CreateAuditLog(ctx context.Context, audit *model.AuditLog) error {
	return r.db.Create(ctx, audit)
}

func (r *repository) ListAuditLogs(ctx context.Context, orgID int64, filter AuditLogFilter) ([]model.AuditLog, error) {
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	opts := []database.QueryOption{
		database.Where("org_id = ?", orgID),
		database.Order("created_at DESC, id DESC"),
		database.Limit(filter.Limit),
	}
	if filter.Action != "" {
		opts = append(opts, database.Where("action = ?", filter.Action))
	}
	if filter.UserID > 0 {
		opts = append(opts, database.Where("user_id = ?", filter.UserID))
	}
	if !filter.From.IsZero() {
		opts = append(opts, database.Where("created_at >= ?", filter.From))
	}
	if !filter.To.IsZero() {
		opts = append(opts, database.Where("created_at <= ?", filter.To))
	}
	if filter.Cursor > 0 {
		opts = append(opts, database.Where("id < ?", filter.Cursor))
	}
	var logs []model.AuditLog
	err := r.db.Find(ctx, &logs, opts...)
	return logs, err
}

func (r *repository) AddCasbinRule(ctx context.Context, rule *model.CasbinRule) error {
	var existing model.CasbinRule
	err := r.db.First(ctx, &existing,
		database.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ? AND v3 = ? AND v4 = ? AND v5 = ?",
			rule.PType, rule.V0, rule.V1, rule.V2, rule.V3, rule.V4, rule.V5),
	)
	if err == nil {
		return nil
	}
	if !errors.Is(err, iamservice.ErrNotFound) {
		return err
	}
	return r.db.Create(ctx, rule)
}

func (r *repository) DeleteCasbinRules(ctx context.Context, ptype string, values ...string) error {
	opts := []database.QueryOption{database.Where("ptype = ?", ptype)}
	for i, value := range values {
		if value == "" {
			continue
		}
		opts = append(opts, database.Where("v"+strconv.Itoa(i)+" = ?", value))
	}
	_, err := r.db.Delete(ctx, &model.CasbinRule{}, opts...)
	return err
}

func (r *repository) ListCasbinRules(ctx context.Context) ([]iamservice.AuthorizationRule, error) {
	var rows []model.CasbinRule
	if err := r.db.Find(ctx, &rows, database.Order("id ASC")); err != nil {
		return nil, err
	}
	rules := make([]iamservice.AuthorizationRule, 0, len(rows))
	for _, row := range rows {
		values := []string{row.V0, row.V1, row.V2, row.V3, row.V4, row.V5}
		switch row.PType {
		case "p":
			values = values[:6]
		case "g":
			values = values[:3]
		}
		rules = append(rules, iamservice.AuthorizationRule{PType: row.PType, Values: values})
	}
	return rules, nil
}

func alive() database.QueryOption {
	return database.Where("deleted_at IS NULL")
}
