package sync

import (
	"net/http"

	"github.com/ChenJustin666/axiops/application/manager-api/internal/logic/sync"
	"github.com/ChenJustin666/axiops/application/manager-api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ClusterAllSyncHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := sync.NewClusterAllSyncLogic(r.Context(), svcCtx)
		resp, err := l.ClusterAllSync()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
