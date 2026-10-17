package storage

import (
	"net/http"

	"github.com/ChenJustin666/axiops/application/portal-api/internal/logic/storage"
	"github.com/ChenJustin666/axiops/application/portal-api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadImageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := storage.NewUploadImageLogic(r.Context(), svcCtx)
		resp, err := l.UploadImage(r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
