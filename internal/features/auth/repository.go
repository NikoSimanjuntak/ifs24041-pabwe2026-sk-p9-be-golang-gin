package auth

import (
	"app/internal/database"

	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(u *database.User) error { return r.db.Create(u).Error }

func (r *Repository) Update(u *database.User) error { return r.db.Save(u).Error }

func (r *Repository) FindByID(id uint) (*database.User, error) {
	var u database.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByLogin mencari pengguna berdasarkan username atau email.
func (r *Repository) FindByLogin(login string) (*database.User, error) {
	var u database.User
	if err := r.db.Where("username = ? OR email = ?", login, login).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Taken memeriksa apakah username/email sudah dipakai pengguna lain (selain excludeID).
func (r *Repository) Taken(username, email string, excludeID uint) (bool, error) {
	var n int64
	err := r.db.Model(&database.User{}).
		Where("(username = ? OR email = ?) AND id <> ?", username, email, excludeID).
		Count(&n).Error
	return n > 0, err
}
