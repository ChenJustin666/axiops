package workflow

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
)

func productionResponse(w http.ResponseWriter, result interface{}, err error) {
	if err != nil {
		code := errcode.CodeParamInvalid
		var dbErr *mysql.MySQLError
		var networkErr net.Error
		if errors.As(err, &dbErr) || errors.As(err, &networkErr) || errors.Is(err, driver.ErrBadConn) || errors.Is(err, sql.ErrConnDone) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			code = errcode.CodeDBError
		}
		response.ErrorWithMsg(w, code, err.Error())
		return
	}
	response.Success(w, result)
}
func (h *Handler) GetProductionPolicy(w http.ResponseWriter, r *http.Request) {
	result, err := h.logic.ProductionPolicy(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), nil)
	productionResponse(w, result, err)
}
func (h *Handler) SaveProductionPolicy(w http.ResponseWriter, r *http.Request) {
	var input ProductionPolicy
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&input) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	result, err := h.logic.ProductionPolicy(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), &input)
	productionResponse(w, result, err)
}
func (h *Handler) VerificationOptions(w http.ResponseWriter, r *http.Request) {
	result, err := h.logic.VerificationOptions(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"))
	productionResponse(w, result, err)
}
func (h *Handler) SaveVerification(w http.ResponseWriter, r *http.Request) {
	var input VerificationInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&input) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	result, err := h.logic.SaveVerification(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), input)
	productionResponse(w, result, err)
}
func (h *Handler) ProductionOptions(w http.ResponseWriter, r *http.Request) {
	result, err := h.logic.ProductionOptions(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"))
	productionResponse(w, result, err)
}
func (h *Handler) ProductionPreview(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Has("build_id") {
		id, err := strconv.ParseInt(r.URL.Query().Get("build_id"), 10, 64)
		if err != nil || id <= 0 || r.URL.Query().Has("verification_id") {
			response.Error(w, errcode.CodeParamInvalid)
			return
		}
		result, err := h.logic.ProductionImagePreview(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), id)
		productionResponse(w, result, err)
		return
	}
	id, _ := strconv.ParseInt(r.URL.Query().Get("verification_id"), 10, 64)
	result, err := h.logic.ProductionPreview(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), id)
	productionResponse(w, result, err)
}
func (h *Handler) SubmitProduction(w http.ResponseWriter, r *http.Request) {
	var input ProductionSubmission
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&input) != nil {
		response.Error(w, errcode.CodeParamInvalid)
		return
	}
	input.RequestKey = r.Header.Get("Idempotency-Key")
	id, err := h.logic.SubmitProduction(r.Context(), middleware.GetUserID(r.Context()), pathID(r, "id"), input)
	productionResponse(w, map[string]int64{"workflow_run_id": id}, err)
}
