package response

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Response 统一响应结构
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id"`
}

// PageData 分页数据
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	setJSONCharset(w)
	resp := &Response{
		Code:      errcode.CodeSuccess,
		Message:   "success",
		Data:      data,
		RequestID: generateRequestID(),
	}
	httpx.OkJsonCtx(nil, w, resp)
}

// SuccessWithPage 成功分页响应
func SuccessWithPage(w http.ResponseWriter, list interface{}, total int64, page, pageSize int) {
	setJSONCharset(w)
	resp := &Response{
		Code:    errcode.CodeSuccess,
		Message: "success",
		Data: &PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
		RequestID: generateRequestID(),
	}
	httpx.OkJsonCtx(nil, w, resp)
}

// Error 错误响应
func Error(w http.ResponseWriter, code int) {
	setJSONCharset(w)
	resp := &Response{
		Code:      code,
		Message:   errcode.Message(code),
		RequestID: generateRequestID(),
	}
	// 根据错误码决定 HTTP 状态码
	httpStatus := mapHTTPStatus(code)
	httpx.WriteJsonCtx(nil, w, httpStatus, resp)
}

// ErrorWithMsg 自定义消息的错误响应
func ErrorWithMsg(w http.ResponseWriter, code int, msg string) {
	setJSONCharset(w)
	resp := &Response{
		Code:      code,
		Message:   msg,
		RequestID: generateRequestID(),
	}
	httpStatus := mapHTTPStatus(code)
	httpx.WriteJsonCtx(nil, w, httpStatus, resp)
}

func setJSONCharset(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

// mapHTTPStatus 根据业务错误码映射 HTTP 状态码
func mapHTTPStatus(code int) int {
	switch {
	case code == 0:
		return http.StatusOK
	case code >= 10001 && code <= 10002:
		return http.StatusUnauthorized
	case code == 10003:
		return http.StatusForbidden
	case code >= 10004 && code <= 10006:
		return http.StatusUnauthorized
	case code == 50004:
		return http.StatusBadRequest
	case code == 50005:
		return http.StatusNotFound
	case code >= 50001:
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

func generateRequestID() string {
	return uuid.New().String()
}
