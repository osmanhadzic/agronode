package tenancy

import "context"

type contextKey string

const organizationIDContextKey contextKey = "organization_id"
const userRoleContextKey contextKey = "user_role"

func WithOrganizationID(ctx context.Context, organizationID uint) context.Context {
	if organizationID == 0 {
		return ctx
	}

	return context.WithValue(ctx, organizationIDContextKey, organizationID)
}

func OrganizationIDFromContext(ctx context.Context) (uint, bool) {
	if ctx == nil {
		return 0, false
	}

	value := ctx.Value(organizationIDContextKey)
	if value == nil {
		return 0, false
	}

	organizationID, ok := value.(uint)
	if !ok || organizationID == 0 {
		return 0, false
	}

	return organizationID, true
}

func WithUserRole(ctx context.Context, role string) context.Context {
	if ctx == nil || role == "" {
		return ctx
	}

	return context.WithValue(ctx, userRoleContextKey, role)
}

func UserRoleFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	value := ctx.Value(userRoleContextKey)
	if value == nil {
		return "", false
	}

	role, ok := value.(string)
	if !ok || role == "" {
		return "", false
	}

	return role, true
}
