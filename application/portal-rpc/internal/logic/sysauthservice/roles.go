package sysauthservicelogic

import (
	"strings"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/model"
)

type roleLookup func(roleID uint64) (*model.SysRole, error)

func resolveUserRoleCodes(assignments []*model.SysUserRole, lookup roleLookup) []string {
	roleCodes := make([]string, 0, len(assignments))
	seen := make(map[string]struct{}, len(assignments))

	for _, assignment := range assignments {
		if assignment == nil || assignment.IsDeleted != 0 {
			continue
		}

		role, err := lookup(assignment.RoleId)
		if err != nil || role == nil || role.IsDeleted != 0 {
			continue
		}

		code := strings.TrimSpace(role.Code)
		if code == "" {
			continue
		}

		key := strings.ToUpper(code)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		roleCodes = append(roleCodes, code)
	}

	return roleCodes
}
