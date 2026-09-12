package tenancy

import "context"

type contextKey string

const organizationIDContextKey contextKey = "organization_id"

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
