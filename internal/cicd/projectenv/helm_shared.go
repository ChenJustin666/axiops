package projectenv

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
	"net/http"
)

type sharedService struct {
	ServiceID int64  `json:"service_id"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	Changed   bool   `json:"changed"`
}
type sharedValuesRequest struct {
	Values   string           `json:"values"`
	Revision int64            `json:"revision"`
	Tokens   map[int64]string `json:"tokens"`
}

func sharedServices(ctx context.Context, tx *sql.Tx, eid int64) ([]sharedService, error) {
	rows, err := tx.QueryContext(ctx, `SELECT svc.id,svc.name FROM app_service svc JOIN environment_service_override ov ON ov.service_id=svc.id WHERE ov.project_environment_id=? AND svc.delivery_type='helm' AND svc.status='active' ORDER BY svc.id FOR UPDATE`, eid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []sharedService{}
	for rows.Next() {
		var s sharedService
		if err = rows.Scan(&s.ServiceID, &s.Name); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (h *Handler) HelmSharedValues(w http.ResponseWriter, r *http.Request) {
	uid, eid := middleware.GetUserID(r.Context()), pathID(r, "id")
	env, code := h.logic.Get(r.Context(), uid, eid)
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	update := r.Method == http.MethodPost
	if update {
		if code = h.logic.authorizeEnvironment(r.Context(), uid, env, "update"); code != errcode.CodeSuccess {
			response.Error(w, code)
			return
		}
	}
	repo, ok := h.logic.repo.(*Repository)
	if !ok {
		response.Error(w, errcode.CodeDBError)
		return
	}
	var req sharedValuesRequest
	if update {
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil || req.Revision < 0 {
			response.Error(w, errcode.CodeParamInvalid)
			return
		}
		if _, err := helmconfig.ParseValues(req.Values); err != nil {
			response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
			return
		}
	}
	tx, err := repo.db.BeginTx(r.Context(), nil)
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	defer tx.Rollback()
	// Use the same service -> environment lock order as service configuration saves.
	services, err := sharedServices(r.Context(), tx, eid)
	if err == nil {
		var id int64
		err = tx.QueryRowContext(r.Context(), `SELECT id FROM project_environment WHERE id=? AND status='active' FOR UPDATE`, eid).Scan(&id)
	}
	var shared helmconfig.Shared
	if err == nil {
		shared, err = helmconfig.LoadShared(r.Context(), tx, eid)
	}
	if err != nil {
		response.Error(w, errcode.CodeDBError)
		return
	}
	if update && (shared.Revision != req.Revision || len(req.Tokens) != len(services)) {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "环境配置已变化，请重新载入共享 values")
		return
	}
	for i := range services {
		s := &services[i]
		action := "view"
		if update {
			action = "update"
		}
		if _, _, code = h.logic.authorizeEnvironmentService(r.Context(), uid, eid, s.ServiceID, action); code != errcode.CodeSuccess {
			response.Error(w, code)
			return
		}
		c, e := readServiceConfig(r.Context(), tx, eid, s.ServiceID, true)
		if e != nil {
			err = e
			break
		}
		s.Token = c.Token
		if update {
			if req.Tokens[s.ServiceID] != c.Token {
				err = fmt.Errorf("服务配置已变化，请重新载入共享 values")
				break
			}
			_, s.Before, e = workflow.RenderHelmServicePreview(r.Context(), tx, repo.decryptor, env.ProjectID, eid, s.ServiceID, c.RevisionID)
			// Permit repairing a shared value that currently prevents rendering.
			if e != nil {
				s.Before = ""
			}
		}
	}
	if err == nil && update {
		_, err = tx.ExecContext(r.Context(), `INSERT INTO environment_resource(environment_id,name,kind,manifest,revision,created_by) VALUES(?,?,?,?,1,?) ON DUPLICATE KEY UPDATE manifest=VALUES(manifest),revision=revision+1`, eid, helmconfig.SharedName, helmconfig.SharedKind, req.Values, uid)
		for i := range services {
			if err != nil {
				break
			}
			s := &services[i]
			c, e := readServiceConfig(r.Context(), tx, eid, s.ServiceID, true)
			if e != nil {
				err = e
				break
			}
			_, s.After, e = workflow.RenderHelmServicePreview(r.Context(), tx, repo.decryptor, env.ProjectID, eid, s.ServiceID, c.RevisionID)
			if e != nil {
				err = fmt.Errorf("服务 %s 的共享 values 校验失败；未保存任何配置", s.Name)
				break
			}
			s.Changed = s.Before != s.After
		}
	}
	if err != nil {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, err.Error())
		return
	}
	saved := update && r.URL.Query().Get("preview") != "1"
	if saved {
		if err = tx.Commit(); err != nil {
			response.Error(w, errcode.CodeDBError)
			return
		}
		shared.Values, shared.Revision = req.Values, shared.Revision+1
	}
	response.Success(w, map[string]interface{}{"values": shared.Values, "revision": shared.Revision, "services": services, "saved": saved, "validated": update})
}
