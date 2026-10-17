package credential

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

// Logic 凭据业务逻辑
type Logic struct {
	repo       *Repository
	authorizer authz.Checker
	box        *secretbox.Box
}

func NewLogic(db *sql.DB, authorizer authz.Checker, encryptionKey ...string) *Logic {
	logic := &Logic{repo: NewRepository(db), authorizer: authorizer}
	if len(encryptionKey) > 0 && encryptionKey[0] != "" {
		logic.box, _ = secretbox.New(encryptionKey[0])
	}
	return logic
}

// List 凭据列表
func (l *Logic) List(ctx context.Context, userID int64, credType string, projectID int64) ([]Credential, int) {
	scope, err := l.authorizer.Scopes(ctx, userID, "credential", "view")
	if err != nil {
		return nil, errcode.CodeDBError
	}
	creds, err := l.repo.List(ctx, scope, credType, projectID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	for i := range creds {
		l.describeSecret(&creds[i])
	}
	return creds, errcode.CodeSuccess
}

// GetByID 凭据详情
func (l *Logic) GetByID(ctx context.Context, userID, id int64) (*Credential, int) {
	c, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if c == nil {
		return nil, errcode.CodeCredentialNotFound
	}
	if code := l.authorize(ctx, userID, "view", c.ProjectID); code != errcode.CodeSuccess {
		return nil, code
	}
	l.describeSecret(c)
	return c, errcode.CodeSuccess
}

// Create 创建凭据
func (l *Logic) Create(ctx context.Context, req *CreateCredentialRequest, createdBy int64) (int64, int) {
	if code := l.authorize(ctx, createdBy, "create", req.ProjectID); code != errcode.CodeSuccess {
		return 0, code
	}
	if l.box == nil {
		log.Printf("credential create rejected: encryption box unavailable user=%d project=%d", createdBy, req.ProjectID)
		return 0, errcode.CodeInternalError
	}
	if !json.Valid([]byte(req.Data)) {
		return 0, errcode.CodeParamInvalid
	}
	if req.Type == "secret" {
		if _, err := secretFields(req.Data); err != nil {
			return 0, errcode.CodeParamInvalid
		}
	}
	encrypted, err := l.box.Encrypt(req.Data)
	if err != nil {
		log.Printf("credential encryption failed user=%d project=%d type=%q: %v", createdBy, req.ProjectID, req.Type, err)
		return 0, errcode.CodeInternalError
	}
	c := &Credential{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Data:        encrypted,
		ProjectID:   req.ProjectID,
		CreatedBy:   createdBy,
	}

	id, err := l.repo.Create(ctx, c)
	if err != nil {
		log.Printf("credential database insert failed user=%d project=%d type=%q: %v", createdBy, req.ProjectID, req.Type, err)
		return 0, errcode.CodeDBError
	}
	return id, errcode.CodeSuccess
}

// Update 更新凭据
func (l *Logic) Update(ctx context.Context, userID, id int64, req *UpdateCredentialRequest) int {
	c, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if c == nil {
		return errcode.CodeCredentialNotFound
	}
	if code := l.authorize(ctx, userID, "update", c.ProjectID); code != errcode.CodeSuccess {
		return code
	}

	if req.Name != "" {
		c.Name = req.Name
	}
	if req.Description != "" {
		c.Description = req.Description
	}
	if req.Data != "" {
		if l.box == nil {
			return errcode.CodeInternalError
		}
		if !json.Valid([]byte(req.Data)) {
			return errcode.CodeParamInvalid
		}
		if c.Type == "secret" {
			if _, err := secretFields(req.Data); err != nil {
				return errcode.CodeParamInvalid
			}
		}
		encrypted, err := l.box.Encrypt(req.Data)
		if err != nil {
			return errcode.CodeInternalError
		}
		c.Data = encrypted
	} else if !strings.HasPrefix(c.Data, "v1:") {
		// Do not silently preserve legacy plaintext when another field is edited.
		return errcode.CodeInternalError
	}

	if err := l.repo.Update(ctx, c); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) DeriveSecret(ctx context.Context, userID, id int64, req *DeriveSecretRequest) (int64, int) {
	tx, err := l.repo.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	defer tx.Rollback()
	var c Credential
	err = tx.QueryRowContext(ctx, `SELECT type,project_id,data,description FROM credential WHERE id=? FOR UPDATE`, id).Scan(&c.Type, &c.ProjectID, &c.Data, &c.Description)
	if err == sql.ErrNoRows {
		return 0, errcode.CodeCredentialNotFound
	}
	if err != nil {
		return 0, errcode.CodeDBError
	}
	for _, action := range []string{"view", "update", "create"} {
		if code := l.authorize(ctx, userID, action, c.ProjectID); code != 0 {
			return 0, code
		}
	}
	if c.Type != "secret" || len(req.Token) != 64 || req.Token != credentialToken(c.Data) {
		return 0, errcode.CodeParamInvalid
	}
	if l.box == nil {
		return 0, errcode.CodeInternalError
	}
	plain, err := l.box.Decrypt(c.Data)
	if err != nil {
		return 0, errcode.CodeInternalError
	}
	source, err := secretFields(plain)
	if err != nil {
		return 0, errcode.CodeParamInvalid
	}
	next, err := mergeSecretFields(source, req)
	if err != nil {
		return 0, errcode.CodeParamInvalid
	}
	encrypted, err := l.box.Encrypt(next)
	if err != nil {
		return 0, errcode.CodeInternalError
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO credential(name,type,description,data,project_id,created_by) VALUES(?,'secret',?,?,?,?)`, strings.TrimSpace(req.Name), c.Description, encrypted, c.ProjectID, userID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	nextID, err := result.LastInsertId()
	if err != nil || tx.Commit() != nil {
		return 0, errcode.CodeDBError
	}
	return nextID, errcode.CodeSuccess
}

// Delete 删除凭据
func (l *Logic) Delete(ctx context.Context, userID, id int64) int {
	c, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if c == nil {
		return errcode.CodeCredentialNotFound
	}
	if code := l.authorize(ctx, userID, "delete", c.ProjectID); code != errcode.CodeSuccess {
		return code
	}
	if err := l.repo.Delete(ctx, id); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) authorize(ctx context.Context, userID int64, action string, projectID int64) int {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "credential", Action: action, ProjectID: projectID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}
