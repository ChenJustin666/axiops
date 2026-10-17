package sysauthservicelogic

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/model"
)

func TestResolveUserRolesEmptyAssignmentsDefaultDeny(t *testing.T) {
	called := false
	codes := resolveUserRoleCodes(nil, func(roleID uint64) (*model.SysRole, error) {
		called = true
		return nil, nil
	})

	if called {
		t.Fatal("role lookup must not run without assignments")
	}
	if len(codes) != 0 {
		t.Fatalf("expected no roles, got codes=%v", codes)
	}
}

func TestResolveUserRolesSkipsInvalidAndDeduplicates(t *testing.T) {
	assignments := []*model.SysUserRole{
		{RoleId: 1},
		{RoleId: 2},
		{RoleId: 3},
		{RoleId: 1},
		nil,
	}
	roles := map[uint64]*model.SysRole{
		1: {Id: 1, Code: "viewer"},
		2: {Id: 2, Code: "  "},
	}

	codes := resolveUserRoleCodes(assignments, func(roleID uint64) (*model.SysRole, error) {
		if roleID == 3 {
			return nil, errors.New("stale role binding")
		}
		return roles[roleID], nil
	})

	if !reflect.DeepEqual(codes, []string{"viewer"}) {
		t.Fatalf("unexpected role codes: %v", codes)
	}
}

func TestResolveUserRolesNeverInjectsLegacySuperRole(t *testing.T) {
	codes := resolveUserRoleCodes([]*model.SysUserRole{{RoleId: 99}}, func(roleID uint64) (*model.SysRole, error) {
		return nil, errors.New("role not found")
	})

	for _, code := range codes {
		if code == "R_SUPER" || code == "SUPER_ADMIN" {
			t.Fatalf("unexpected elevated role: %s", code)
		}
	}
}

func TestResolveUserRolesIgnoresDeletedAssignmentsAndRoles(t *testing.T) {
	codes := resolveUserRoleCodes([]*model.SysUserRole{{RoleId: 1, IsDeleted: 1}, {RoleId: 2}, {RoleId: 3}}, func(id uint64) (*model.SysRole, error) {
		if id == 1 {
			t.Fatal("deleted binding must not resolve a role")
		}
		if id == 2 {
			return &model.SysRole{Code: "SUPER_ADMIN", IsDeleted: 1}, nil
		}
		return &model.SysRole{Code: "viewer"}, nil
	})
	if !reflect.DeepEqual(codes, []string{"viewer"}) {
		t.Fatalf("deleted grants included: %v", codes)
	}
}
