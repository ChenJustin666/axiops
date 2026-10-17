package response

import (
	"net/http"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

func TestMapHTTPStatusOrdersSpecificSystemCodesBeforeInternalErrors(t *testing.T) {
	tests := []struct {
		code int
		want int
	}{
		{errcode.CodeParamInvalid, http.StatusBadRequest},
		{errcode.CodeNotFound, http.StatusNotFound},
		{errcode.CodeInternalError, http.StatusInternalServerError},
		{errcode.CodeDBError, http.StatusInternalServerError},
	}
	for _, test := range tests {
		if got := mapHTTPStatus(test.code); got != test.want {
			t.Fatalf("mapHTTPStatus(%d)=%d want %d", test.code, got, test.want)
		}
	}
}
