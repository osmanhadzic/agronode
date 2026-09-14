package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"agronode/backend/internal/tenancy"
	"github.com/gin-gonic/gin"
)

const organizationIDQueryParam = "organizationId"
const organizationIDHeader = "X-Organization-ID"

func requestContextWithOrganizationScope(ctx *gin.Context) (context.Context, error) {
	requestContext := ctx.Request.Context()
	if organizationID, ok := tenancy.OrganizationIDFromContext(requestContext); ok {
		return tenancy.WithOrganizationID(requestContext, organizationID), nil
	}

	queryValue := strings.TrimSpace(ctx.Query(organizationIDQueryParam))
	headerValue := strings.TrimSpace(ctx.GetHeader(organizationIDHeader))

	if queryValue == "" && headerValue == "" {
		return nil, fmt.Errorf("organizationId is required")
	}

	queryID, hasQuery, parseErr := parseOrganizationID(queryValue)
	if parseErr != nil {
		return nil, parseErr
	}

	headerID, hasHeader, parseErr := parseOrganizationID(headerValue)
	if parseErr != nil {
		return nil, parseErr
	}

	if hasQuery && hasHeader && queryID != headerID {
		return nil, fmt.Errorf("organization scope mismatch between query and header")
	}

	organizationID := uint(0)
	if hasQuery {
		organizationID = queryID
	} else {
		organizationID = headerID
	}

	return tenancy.WithOrganizationID(requestContext, organizationID), nil
}

func parseOrganizationID(raw string) (organizationID uint, hasValue bool, err error) {
	if strings.TrimSpace(raw) == "" {
		return 0, false, nil
	}

	parsed, parseErr := strconv.ParseUint(raw, 10, 64)
	if parseErr != nil || parsed == 0 {
		return 0, false, fmt.Errorf("organizationId must be a positive integer")
	}

	return uint(parsed), true, nil
}
