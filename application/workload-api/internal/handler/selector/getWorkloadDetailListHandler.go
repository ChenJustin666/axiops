// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package selector

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"github.com/ChenJustin666/axiops/application/workload-api/internal/logic/selector"
	"github.com/ChenJustin666/axiops/application/workload-api/internal/svc"
	"github.com/ChenJustin666/axiops/application/workload-api/internal/types"
	"github.com/ChenJustin666/axiops/common/handler/errorx"
	"github.com/ChenJustin666/axiops/common/verify"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// get workload detail list (environment view)
func GetWorkloadDetailListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WorkloadDetailListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		defaults.SetDefaults(&req)
		if err := svcCtx.Validator.Validate.StructCtx(r.Context(), &req); err != nil {
			strErr := verify.RemoveTopSaStr(err.(validator.ValidationErrors), svcCtx.Validator.Translator)
			httpx.ErrorCtx(r.Context(), w, errorx.New(40020, strErr))
			return
		}
		l := selector.NewGetWorkloadDetailListLogic(r.Context(), svcCtx)
		resp, err := l.GetWorkloadDetailList(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
