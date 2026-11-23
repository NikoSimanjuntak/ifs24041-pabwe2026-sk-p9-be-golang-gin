package ai

import (
	"errors"
	"net/http"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Topik artikel wajib diisi (maks. 300 karakter)")
		return
	}
	res, err := h.svc.Generate(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCategory) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusBadGateway, "Gagal menghasilkan artikel dengan AI: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Draf artikel berhasil dihasilkan", res)
}
