package post

import (
	"strings"

	"app/internal/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(p *database.Post) error {
	return r.db.Omit(clause.Associations).Create(p).Error
}

func (r *Repository) Save(p *database.Post) error {
	return r.db.Omit(clause.Associations).Save(p).Error
}

func (r *Repository) Delete(p *database.Post) error { return r.db.Delete(p).Error }

func (r *Repository) SlugExists(slug string) (bool, error) {
	var n int64
	err := r.db.Model(&database.Post{}).Where("slug = ?", slug).Count(&n).Error
	return n > 0, err
}

func (r *Repository) FindByIDAndUser(id, userID uint) (*database.Post, error) {
	var p database.Post
	if err := r.db.Preload("User").Where("id = ? AND user_id = ?", id, userID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) FindPublicBySlug(slug string) (*database.Post, error) {
	var p database.Post
	if err := r.db.Preload("User").Where("slug = ? AND is_published = ?", slug, true).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) ListPublic(f ListFilter) ([]database.Post, error) {
	q := r.db.Preload("User").Where("is_published = ?", true)
	if f.Query != "" {
		like := "%" + strings.ToLower(f.Query) + "%"
		q = q.Where("(LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?)", like, like, like)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	var posts []database.Post
	err := q.Order("published_at DESC, id DESC").Find(&posts).Error
	return posts, err
}

func (r *Repository) ListByUser(userID uint) ([]database.Post, error) {
	var posts []database.Post
	err := r.db.Preload("User").Where("user_id = ?", userID).Order("created_at DESC, id DESC").Find(&posts).Error
	return posts, err
}

func (r *Repository) ListAll() ([]database.Post, error) {
	var posts []database.Post
	err := r.db.Preload("User").Order("created_at DESC, id DESC").Find(&posts).Error
	return posts, err
}

// CountByUser menghitung total dan jumlah artikel terbit dalam satu query.
func (r *Repository) CountByUser(userID uint) (total, published int64, err error) {
	var row struct{ Total, Published int64 }
	err = r.db.Model(&database.Post{}).
		Select("COUNT(*) AS total, COALESCE(SUM(CASE WHEN is_published THEN 1 ELSE 0 END), 0) AS published").
		Where("user_id = ?", userID).
		Scan(&row).Error
	return row.Total, row.Published, err
}

func (r *Repository) UserExists(id uint) (bool, error) {
	var n int64
	err := r.db.Model(&database.User{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *Repository) UsersWithCounts() ([]AdminUser, error) {
	var rows []AdminUser
	err := r.db.Table("users").
		Select("users.id, users.username, users.email, users.role, users.created_at, " +
			"COUNT(posts.id) AS total_posts, " +
			"COALESCE(SUM(CASE WHEN posts.is_published THEN 1 ELSE 0 END), 0) AS total_published").
		Joins("LEFT JOIN posts ON posts.user_id = users.id").
		Group("users.id").
		Order("users.id").
		Scan(&rows).Error
	return rows, err
}
