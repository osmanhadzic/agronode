package repositories

import (
	"context"

	"agronode/backend/internal/tenancy"
)

func organizationIDFromContext(ctx context.Context) (uint, bool) {
	return tenancy.OrganizationIDFromContext(ctx)
}
