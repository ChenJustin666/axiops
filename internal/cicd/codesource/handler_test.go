package codesource

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/pkg/gitprovider"
)

func TestWriteProviderErrorReturnsActionableAuthenticationMessage(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeProviderError(recorder, gitprovider.ErrUnauthorized)

	if recorder.Code != 400 {
		t.Fatalf("status=%d, want 400", recorder.Code)
	}
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 30009 || !strings.Contains(body.Message, "Token") || !strings.Contains(body.Message, "仓库的读取权限") {
		t.Fatalf("body=%+v, want actionable code-source authentication error", body)
	}
}
