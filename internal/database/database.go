package database

import (
	"errors"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Categories adalah daftar kategori artikel yang diizinkan.
var Categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

// IsValidCategory memeriksa apakah kategori termasuk daftar yang diizinkan.
func IsValidCategory(c string) bool {
	for _, v := range Categories {
		if v == c {
			return true
		}
	}
	return false
}

// User adalah model tabel users.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	Role      string    `gorm:"size:20;not null" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Post adalah model tabel posts.
type Post struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	User           User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Title          string     `gorm:"size:255;not null" json:"title"`
	Slug           string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"size:100" json:"category"`
	Excerpt        string     `gorm:"type:text" json:"excerpt"`
	Content        string     `gorm:"type:text" json:"content"`
	CoverImagePath string     `gorm:"type:text" json:"cover_image_path"`
	IsPublished    bool       `gorm:"not null" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Open membuka koneksi GORM dari dialector apa pun.
func Open(d gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(d, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
}

// ConnectPostgres membuka koneksi PostgreSQL untuk runtime.
func ConnectPostgres(dsn string) (*gorm.DB, error) {
	return Open(postgres.Open(dsn))
}

// OpenSQLiteMemory membuka SQLite in-memory (satu koneksi) khusus untuk pengujian.
func OpenSQLiteMemory() *gorm.DB {
	db, _ := Open(sqlite.Open(":memory:"))
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	return db
}

// Migrate membuat/menyesuaikan tabel users dan posts.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Post{})
}

// Seed memastikan akun admin sesuai konfigurasi env: dibuat bila belum ada,
// bila sudah ada maka role dijadikan admin dan password disamakan dengan env.
func Seed(db *gorm.DB, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	var u User
	err = db.Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&User{
			Username: username,
			Email:    username + "@delcom.local",
			Password: string(hash),
			Role:     RoleAdmin,
		}).Error
	}
	if err != nil {
		return err
	}
	u.Password = string(hash)
	u.Role = RoleAdmin
	return db.Save(&u).Error
}
