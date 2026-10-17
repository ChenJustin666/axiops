package codesource

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/pkg/gitprovider"
)

func TestProviderErrorDistinguishesRepositoryAccessAndLocalPermissions(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{fmt.Errorf("page 2: %w", gitprovider.ErrUnauthorized), "读取权限"},
		{gitprovider.ErrNotFound, "私有仓库"},
		{gitprovider.ErrInvalidResponse, "分页"},
		{gitprovider.ErrNetworkTimeout, "网络"},
	} {
		w := httptest.NewRecorder()
		writeProviderError(w, tc.err)
		if !strings.Contains(w.Body.String(), tc.want) {
			t.Fatalf("missing diagnostic %q: %s", tc.want, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	writeProviderError(w, ErrNoPermission)
	if strings.Contains(w.Body.String(), "Token") {
		t.Fatal("AxiOps permission failure misreported as Git authentication")
	}
}
