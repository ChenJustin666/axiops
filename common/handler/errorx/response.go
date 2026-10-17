package errorx

import (
	"net/http"

	"github.com/ChenJustin666/axiops/common/handler/errorx/types"
)

func ErrHandler(err error) (int, any) {
	code := CodeFromError(err)
	return http.StatusOK, types.Status{
		Code:    int32(code.Code()),
		Message: code.Message(),
	}
}
