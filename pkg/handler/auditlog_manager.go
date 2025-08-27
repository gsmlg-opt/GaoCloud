package handler

import (
	"fmt"

	"pkg/auditlog"

	resterr "gorest/error"
	"gorest/resource"
)

type AuditLogManager struct {
	audit *auditlog.AuditLogger
}

func newAuditLogManager(audit *auditlog.AuditLogger) *AuditLogManager {
	return &AuditLogManager{
		audit: audit,
	}
}

func (a *AuditLogManager) List(ctx *resource.Context) (interface{}, *resterr.APIError) {
	logs, err := a.audit.List(getCurrentUser(ctx))
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("list audit log failed %s", err.Error()))
	}
	return logs, nil
}
