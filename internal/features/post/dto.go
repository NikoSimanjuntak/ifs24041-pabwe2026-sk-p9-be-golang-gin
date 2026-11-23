package post

import (
	"time"

	"app/internal/database"
)

// PostRequest dipakai untuk membuat maupun memperbarui artikel.
// IsPublished bertipe pointer: nil berarti "tidak diubah" saat update dan "draf" saat create.
type PostRequest struct {
	Title          string `json:"title" binding:"required,max=255"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    *bool  `json:"is_published"`
}

type ListFilter struct {
	Query    string
	Category string
}

type AuthorInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type PostResponse struct {
	ID             uint        `json:"id"`
	UserID         uint        `json:"user_id"`
	Title          string      `json:"title"`
	Slug           string      `json:"slug"`
	Category       string      `json:"category"`
	Excerpt        string      `json:"excerpt"`
	Content        string      `json:"content"`
	CoverImagePath string      `json:"cover_image_path"`
	IsPublished    bool        `json:"is_published"`
	PublishedAt    *time.Time  `json:"published_at"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	ReadingTime    int         `json:"reading_time"`
	Author         *AuthorInfo `json:"author,omitempty"`
}

type Stats struct {
	Total     int64 `json:"total_posts"`
	Published int64 `json:"total_published"`
	Draft     int64 `json:"total_draft"`
}

type AdminUser struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	TotalPosts     int64     `json:"total_posts"`
	TotalPublished int64     `json:"total_published"`
}

func toResponse(p database.Post) PostResponse {
	r := PostResponse{
		ID: p.ID, UserID: p.UserID, Title: p.Title, Slug: p.Slug, Category: p.Category,
		Excerpt: p.Excerpt, Content: p.Content, CoverImagePath: p.CoverImagePath,
		IsPublished: p.IsPublished, PublishedAt: p.PublishedAt,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		ReadingTime: ReadingTime(p.Content),
	}
	if p.User.ID != 0 {
		r.Author = &AuthorInfo{ID: p.User.ID, Username: p.User.Username}
	}
	return r
}

func toResponses(posts []database.Post) []PostResponse {
	out := make([]PostResponse, 0, len(posts))
	for _, p := range posts {
		out = append(out, toResponse(p))
	}
	return out
}
