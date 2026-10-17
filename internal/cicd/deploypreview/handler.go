package deploypreview

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

type Handler struct{ logic *Logic }

func NewHandler(logic *Logic) *Handler { return &Handler{logic: logic} }

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	environmentID := positiveInt64(r.PathValue("id"))
	serviceID := positiveInt64(r.URL.Query().Get("service_id"))
	revisionID := optionalInt64(r.URL.Query().Get("service_revision_id"))
	if environmentID == 0 || serviceID == 0 || revisionID < 0 {
		response.ErrorWithMsg(w, errcode.CodeParamInvalid, "无效的环境、服务或服务版本")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	result, code := h.logic.Preview(ctx, middleware.GetUserID(r.Context()), environmentID, Request{
		ServiceID: serviceID, ServiceRevisionID: revisionID,
		ReleaseName: strings.TrimSpace(r.URL.Query().Get("release_name")),
	})
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, result)
}

func positiveInt64(value string) int64 {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

func optionalInt64(value string) int64 {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return -1
	}
	return id
}
