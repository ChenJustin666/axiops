package release

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLegacyEndpointsNeverReadOrWriteReleaseData(t *testing.T) {
	h := NewHandler(nil)
	for _, handle := range []http.HandlerFunc{h.ListByService, h.GetByID, h.Create, h.Approve, h.Rollback} {
		w := httptest.NewRecorder()
		handle(w, httptest.NewRequest(http.MethodPost, "/api/v1/releases", nil))
		if w.Code != http.StatusGone {
			t.Fatalf("legacy endpoint returned %d", w.Code)
		}
	}
}
