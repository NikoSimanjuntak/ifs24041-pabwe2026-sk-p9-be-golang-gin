package auth

import (
	"errors"
	"time"

	"app/internal/database"
	"app/internal/middleware"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrConflict           = errors.New("username atau email sudah digunakan")
	ErrInvalidCredentials = errors.New("username atau password salah")
	ErrNotFound           = errors.New("pengguna tidak ditemukan")
)

type Service struct {
	repo   *Repository
	secret string
	ttl    time.Duration
}

func NewService(repo *Repository, secret string, ttl time.Duration) *Service {
	return &Service{repo: repo, secret: secret, ttl: ttl}
}

func (s *Service) Register(req RegisterRequest) (*UserResponse, error) {
	taken, err := s.repo.Taken(req.Username, req.Email, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrConflict
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &database.User{Username: req.Username, Email: req.Email, Password: string(hash), Role: database.RoleMember}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	res := toUser(u)
	return &res, nil
}

func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	u, err := s.repo.FindByLogin(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return &AuthResponse{
		Token: middleware.GenerateToken(s.secret, u.ID, u.Role, s.ttl),
		User:  toUser(u),
	}, nil
}

func (s *Service) Profile(id uint) (*UserResponse, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	res := toUser(u)
	return &res, nil
}

func (s *Service) UpdateProfile(id uint, req UpdateProfileRequest) (*UserResponse, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if req.Username != "" {
		u.Username = req.Username
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	taken, err := s.repo.Taken(u.Username, u.Email, u.ID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrConflict
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u.Password = string(hash)
	}
	if err := s.repo.Update(u); err != nil {
		return nil, err
	}
	res := toUser(u)
	return &res, nil
}
