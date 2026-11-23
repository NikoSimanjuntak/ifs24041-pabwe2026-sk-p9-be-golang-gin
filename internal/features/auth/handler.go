package auth

import (
	"errors"
	"net/http"

	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrConflict):
		response.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidCredentials):
		response.Fail(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data registrasi tidak valid: "+err.Error())
		return
	}
	user, err := h.svc.Register(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Registrasi berhasil", user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}
	res, err := h.svc.Login(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Login berhasil", res)
}

// Logout bersifat stateless: klien cukup membuang token.
func (h *Handler) Logout(c *gin.Context) {
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

func (h *Handler) Profile(c *gin.Context) {
	user, err := h.svc.Profile(c.GetUint(middleware.CtxUserID))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profil berhasil diambil", user)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data profil tidak valid: "+err.Error())
		return
	}
	user, err := h.svc.UpdateProfile(c.GetUint(middleware.CtxUserID), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profil berhasil diperbarui", user)
}
