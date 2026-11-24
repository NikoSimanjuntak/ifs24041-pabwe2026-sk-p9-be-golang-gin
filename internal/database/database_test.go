package database

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestMigrate(t *testing.T) {
	db := OpenSQLiteMemory()
	require.NoError(t, Migrate(db))
	assert.True(t, db.Migrator().HasTable(&User{}))
	assert.True(t, db.Migrator().HasTable(&Post{}))
}

func TestConnectPostgresError(t *testing.T) {
	_, err := ConnectPostgres("host=127.0.0.1 port=1 user=x password=x dbname=x sslmode=disable connect_timeout=1")
	assert.Error(t, err)
}

func TestSeed(t *testing.T) {
	db := OpenSQLiteMemory()
	require.NoError(t, Migrate(db))
	require.NoError(t, Seed(db, "admin", "admin"))
	require.NoError(t, Seed(db, "admin", "admin")) // idempotent

	var users []User
	require.NoError(t, db.Find(&users).Error)
	require.Len(t, users, 1)
	assert.Equal(t, RoleAdmin, users[0].Role)
	assert.NotEqual(t, "admin", users[0].Password)
}

func TestSeedPromotesAndSyncsExistingUser(t *testing.T) {
	db := OpenSQLiteMemory()
	require.NoError(t, Migrate(db))
	require.NoError(t, db.Create(&User{Username: "admin", Email: "a@x.com", Password: "lama", Role: RoleMember}).Error)

	require.NoError(t, Seed(db, "admin", "rahasia-baru"))

	var u User
	require.NoError(t, db.Where("username = ?", "admin").First(&u).Error)
	assert.Equal(t, RoleAdmin, u.Role)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("rahasia-baru")))
}

func TestSeedQueryError(t *testing.T) {
	db := OpenSQLiteMemory() // tanpa migrasi -> tabel tidak ada
	assert.Error(t, Seed(db, "admin", "admin"))
}

func TestSeedHashError(t *testing.T) {
	db := OpenSQLiteMemory()
	require.NoError(t, Migrate(db))
	assert.Error(t, Seed(db, "admin", strings.Repeat("x", 100)))
}

func TestSeedSaveError(t *testing.T) {
	db := OpenSQLiteMemory()
	require.NoError(t, Migrate(db))
	require.NoError(t, Seed(db, "admin", "admin"))
	_ = db.Callback().Update().Before("gorm:update").Register("fail", func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("boom"))
	})
	assert.Error(t, Seed(db, "admin", "admin"))
}

func TestIsValidCategory(t *testing.T) {
	assert.True(t, IsValidCategory("Teknologi"))
	assert.False(t, IsValidCategory("Olahraga"))
	assert.False(t, IsValidCategory(""))
}
