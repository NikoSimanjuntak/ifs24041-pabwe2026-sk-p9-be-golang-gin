package post

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidCategory):
		response.Fail(c, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return uint(id), true
}

func uid(c *gin.Context) uint { return c.GetUint(middleware.CtxUserID) }

// ---------- Publik ----------

func (h *Handler) ListPublic(c *gin.Context) {
	posts, err := h.svc.ListPublic(ListFilter{
		Query:    strings.TrimSpace(c.Query("q")),
		Category: strings.TrimSpace(c.Query("category")),
	})
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel berhasil diambil", posts)
}

func (h *Handler) GetPublic(c *gin.Context) {
	p, err := h.svc.GetPublic(c.Param("slug"))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel berhasil diambil", p)
}

// ---------- Penulis ----------

func (h *Handler) ListMine(c *gin.Context) {
	posts, err := h.svc.ListMine(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel Anda berhasil diambil", posts)
}

func (h *Handler) Stats(c *gin.Context) {
	s, err := h.svc.Stats(uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Statistik berhasil diambil", s)
}

func (h *Handler) GetMine(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.GetMine(id, uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel berhasil diambil", p)
}

func (h *Handler) Create(c *gin.Context) {
	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data artikel tidak valid: "+err.Error())
		return
	}
	p, err := h.svc.Create(uid(c), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Artikel berhasil dibuat", p)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data artikel tidak valid: "+err.Error())
		return
	}
	p, err := h.svc.Update(id, uid(c), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diperbarui", p)
}

func (h *Handler) TogglePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.TogglePublish(id, uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Status publikasi berhasil diubah", p)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id, uid(c)); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil dihapus", nil)
}

// ---------- Admin ----------

func (h *Handler) AdminUsers(c *gin.Context) {
	users, err := h.svc.AdminUsers()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar pengguna berhasil diambil", users)
}

func (h *Handler) AdminPosts(c *gin.Context) {
	posts, err := h.svc.AdminPosts()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Arsip artikel berhasil diambil", posts)
}

func (h *Handler) AdminUserPosts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	posts, err := h.svc.AdminUserPosts(id)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel pengguna berhasil diambil", posts)
}
