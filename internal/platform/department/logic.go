package department

import (
	"context"
	"database/sql"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type Logic struct {
	repo       *Repository
	authorizer authz.Checker
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{repo: NewRepository(db), authorizer: authorizer}
}

func (l *Logic) List(ctx context.Context, userID int64) ([]Department, int) {
	if code := l.authorize(ctx, userID, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	items, err := l.repo.List(ctx)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return items, errcode.CodeSuccess
}

func (l *Logic) Create(ctx context.Context, userID int64, req *CreateRequest) (int64, int) {
	if code := l.authorize(ctx, userID, "create"); code != errcode.CodeSuccess {
		return 0, code
	}
	if req == nil || !validName(req.Name) || req.ParentID < 0 || req.LeaderUserID < 0 {
		return 0, errcode.CodeParamInvalid
	}
	exists, err := l.repo.Exists(ctx, req.ParentID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if !exists {
		return 0, errcode.CodeParamInvalid
	}
	id, err := l.repo.Create(ctx, &Department{Name: strings.TrimSpace(req.Name), ParentID: req.ParentID,
		SortOrder: req.SortOrder, LeaderUserID: req.LeaderUserID, CreatedBy: userID})
	if err != nil {
		return 0, errcode.CodeDBError
	}
	return id, errcode.CodeSuccess
}

func (l *Logic) Update(ctx context.Context, userID, id int64, req *UpdateRequest) int {
	if code := l.authorize(ctx, userID, "update"); code != errcode.CodeSuccess {
		return code
	}
	if id <= 0 || req == nil || !validName(req.Name) || req.ParentID < 0 || req.LeaderUserID < 0 || req.ParentID == id {
		return errcode.CodeParamInvalid
	}
	exists, err := l.repo.Exists(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if !exists {
		return errcode.CodeNotFound
	}
	exists, err = l.repo.Exists(ctx, req.ParentID)
	if err != nil {
		return errcode.CodeDBError
	}
	if !exists {
		return errcode.CodeParamInvalid
	}
	cyclic, err := l.repo.IsDescendant(ctx, id, req.ParentID)
	if err != nil {
		return errcode.CodeDBError
	}
	if cyclic {
		return errcode.CodeParamInvalid
	}
	if err := l.repo.Update(ctx, &Department{ID: id, Name: strings.TrimSpace(req.Name), ParentID: req.ParentID,
		SortOrder: req.SortOrder, LeaderUserID: req.LeaderUserID}); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) Delete(ctx context.Context, userID, id int64) int {
	if code := l.authorize(ctx, userID, "delete"); code != errcode.CodeSuccess {
		return code
	}
	if id <= 0 {
		return errcode.CodeParamInvalid
	}
	deleted, err := l.repo.Delete(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if !deleted {
		return errcode.CodeParamInvalid
	}
	return errcode.CodeSuccess
}

func (l *Logic) authorize(ctx context.Context, userID int64, action string) int {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "department", Action: action})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

func validName(name string) bool {
	length := len([]rune(strings.TrimSpace(name)))
	return length > 0 && length <= 64
}
