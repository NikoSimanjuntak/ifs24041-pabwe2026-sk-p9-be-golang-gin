package server

import (
	"net/http"
	"time"

	"app/docs"
	"app/internal/config"
	"app/internal/database"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// New merangkai seluruh route API publik, penulis, dan admin.
func New(cfg *config.Config, db *gorm.DB, aiSvc *ai.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())

	authH := auth.NewHandler(auth.NewService(auth.NewRepository(db), cfg.AppKey, 24*time.Hour))
	postH := post.NewHandler(post.NewService(post.NewRepository(db)))
	aiH := ai.NewHandler(aiSvc)

	requireAuth := middleware.Auth(cfg.AppKey)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/api.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(docs.APIJSON))
	})

	api := r.Group("/api")

	pub := api.Group("/public")
	pub.GET("/categories", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "Daftar kategori berhasil diambil", database.Categories)
	})
	pub.GET("/posts", postH.ListPublic)
	pub.GET("/posts/:slug", postH.GetPublic)

	authLimit := middleware.RateLimiter(60, time.Minute)
	ag := api.Group("/auth")
	ag.POST("/register", authLimit, authH.Register)
	ag.POST("/login", authLimit, authH.Login)
	ag.POST("/logout", requireAuth, authH.Logout)
	ag.GET("/profile", requireAuth, authH.Profile)
	ag.PUT("/profile", requireAuth, authH.UpdateProfile)

	posts := api.Group("/posts", requireAuth)
	posts.GET("", postH.ListMine)
	posts.POST("", postH.Create)
	posts.GET("/stats", postH.Stats)
	posts.POST("/ai-generate", middleware.RateLimiter(30, time.Minute), aiH.Generate)
	posts.GET("/:id", postH.GetMine)
	posts.PUT("/:id", postH.Update)
	posts.PATCH("/:id/toggle-publish", postH.TogglePublish)
	posts.DELETE("/:id", postH.Delete)

	admin := api.Group("/admin", requireAuth, middleware.RequireRole(database.RoleAdmin))
	admin.GET("/users", postH.AdminUsers)
	admin.GET("/posts", postH.AdminPosts)
	admin.GET("/users/:id/posts", postH.AdminUserPosts)

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})
	return r
}
