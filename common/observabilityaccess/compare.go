package observabilityaccess

import (
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
)

func comparisonNamespaces(r *http.Request) ([]string, error) {
	var req struct {
		Namespaces []string `form:"namespaces"`
	}
	err := httpx.Parse(r, &req)
	return req.Namespaces, err
}
