package authz

import (
	"fmt"
	"strings"
)

// SQLScopeColumns maps server-owned SQL expressions to Authorize Request fields.
// Never fill these expressions from HTTP input. Missing columns represent the
// same zero/empty fields as a Request; they never mean an arbitrary resource.
type SQLScopeColumns struct {
	ProjectID, ServiceID, BuildID, WorkflowID, ClusterID, EnvironmentID, Environment string
}

func validPolicyScope(p Policy) bool {
	switch p.ScopeType {
	case "", ScopePlatform:
		return true
	case ScopeProject, ScopeService, ScopeBuild, ScopeWorkflow, ScopeCluster, ScopeEnvironment:
		return p.ScopeID > 0
	default:
		return false
	}
}

// Compile the policy conjunctions before combining allows and denies. This
// preserves deny precedence and prevents NULL from undoing a denial in SQL.
func (s ScopeSet) policySQLPredicate(c SQLScopeColumns) (string, []interface{}) {
	var allow, deny []string
	var allowArgs, denyArgs []interface{}
	columns := map[string]string{
		ScopeProject: c.ProjectID, ScopeService: c.ServiceID, ScopeBuild: c.BuildID,
		ScopeWorkflow: c.WorkflowID, ScopeCluster: c.ClusterID, ScopeEnvironment: c.EnvironmentID,
	}
	for _, p := range s.policies {
		if !validPolicyScope(p) {
			continue
		}
		var terms []string
		var args []interface{}
		if p.ScopeType != "" && p.ScopeType != ScopePlatform {
			column := columns[p.ScopeType]
			if column == "" {
				continue
			}
			// MySQL's null-safe equality keeps NULL false for positive scope IDs
			// without wrapping indexed identity columns in a function.
			terms = append(terms, fmt.Sprintf("%s <=> ?", column))
			args = append(args, p.ScopeID)
		}
		if p.Environment != "" && p.Environment != "*" {
			if c.Environment == "" {
				continue
			}
			// Go strings compare case-sensitively; MySQL's default collation does not.
			terms = append(terms, fmt.Sprintf("BINARY %s <=> BINARY ?", c.Environment))
			args = append(args, p.Environment)
		}
		predicate := "1 = 1"
		if len(terms) > 0 {
			predicate = "(" + strings.Join(terms, " AND ") + ")"
		}
		if strings.EqualFold(p.Effect, "deny") {
			deny = append(deny, predicate)
			denyArgs = append(denyArgs, args...)
		} else if strings.EqualFold(p.Effect, "allow") {
			allow = append(allow, predicate)
			allowArgs = append(allowArgs, args...)
		}
	}
	if len(allow) == 0 {
		return "1 = 0", nil
	}
	query := "(" + strings.Join(allow, " OR ") + ")"
	if len(deny) > 0 {
		query += " AND NOT (" + strings.Join(deny, " OR ") + ")"
	}
	return query, append(allowArgs, denyArgs...)
}
