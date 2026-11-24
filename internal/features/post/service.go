package post

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"app/internal/database"

	"gorm.io/gorm"
)

var (
	ErrNotFound        = errors.New("data tidak ditemukan")
	ErrInvalidCategory = errors.New("kategori tidak valid")
	ErrForbidden       = errors.New("anda tidak berhak mengakses artikel ini")
)

const wordsPerMinute = 200

// Slugify mengubah judul menjadi slug URL (huruf kecil, angka, dan tanda hubung).
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return "post"
	}
	return out
}

// ReadingTime menghitung estimasi waktu baca dalam menit (kata / 200 wpm, minimal 1).
func ReadingTime(content string) int {
	t := (len(strings.Fields(content)) + wordsPerMinute - 1) / wordsPerMinute
	if t < 1 {
		return 1
	}
	return t
}

func normalizeCategory(c string) (string, error) {
	c = strings.TrimSpace(c)
	if c == "" {
		return "Umum", nil
	}
	if !database.IsValidCategory(c) {
		return "", ErrInvalidCategory
	}
	return c, nil
}

func mapErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func setPublished(p *database.Post, publish bool) {
	if publish == p.IsPublished {
		return
	}
	p.IsPublished = publish
	if publish {
		now := time.Now()
		p.PublishedAt = &now
	} else {
		p.PublishedAt = nil
	}
}

type Service struct{ repo *Repository }

// owned mengambil artikel dan memastikan milik userID: 404 bila tidak ada, 403 bila milik orang lain.
func (s *Service) owned(id, userID uint) (*database.Post, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, mapErr(err)
	}
	if p.UserID != userID {
		return nil, ErrForbidden
	}
	return p, nil
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) uniqueSlug(title string) (string, error) {
	base := Slugify(title)
	slug := base
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExists(slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func (s *Service) Create(userID uint, req PostRequest) (*PostResponse, error) {
	category, err := normalizeCategory(req.Category)
	if err != nil {
		return nil, err
	}
	slug, err := s.uniqueSlug(req.Title)
	if err != nil {
		return nil, err
	}
	p := &database.Post{
		UserID: userID, Title: strings.TrimSpace(req.Title), Slug: slug, Category: category,
		Excerpt: req.Excerpt, Content: req.Content, CoverImagePath: req.CoverImagePath,
	}
	setPublished(p, req.IsPublished != nil && *req.IsPublished)
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	res := toResponse(*p)
	return &res, nil
}

func (s *Service) Update(id, userID uint, req PostRequest) (*PostResponse, error) {
	p, err := s.owned(id, userID)
	if err != nil {
		return nil, err
	}
	category, err := normalizeCategory(req.Category)
	if err != nil {
		return nil, err
	}
	p.Title = strings.TrimSpace(req.Title)
	p.Category = category
	p.Excerpt = req.Excerpt
	p.Content = req.Content
	p.CoverImagePath = req.CoverImagePath
	if req.IsPublished != nil {
		setPublished(p, *req.IsPublished)
	}
	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	res := toResponse(*p)
	return &res, nil
}

func (s *Service) TogglePublish(id, userID uint) (*PostResponse, error) {
	p, err := s.owned(id, userID)
	if err != nil {
		return nil, err
	}
	setPublished(p, !p.IsPublished)
	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	res := toResponse(*p)
	return &res, nil
}

func (s *Service) Delete(id, userID uint) error {
	p, err := s.owned(id, userID)
	if err != nil {
		return err
	}
	return s.repo.Delete(p)
}

func (s *Service) GetMine(id, userID uint) (*PostResponse, error) {
	p, err := s.owned(id, userID)
	if err != nil {
		return nil, err
	}
	res := toResponse(*p)
	return &res, nil
}

func (s *Service) ListMine(userID uint) ([]PostResponse, error) {
	posts, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	return toResponses(posts), nil
}

func (s *Service) Stats(userID uint) (*Stats, error) {
	total, published, err := s.repo.CountByUser(userID)
	if err != nil {
		return nil, err
	}
	return &Stats{Total: total, Published: published, Draft: total - published}, nil
}

func (s *Service) ListPublic(f ListFilter) ([]PostResponse, error) {
	posts, err := s.repo.ListPublic(f)
	if err != nil {
		return nil, err
	}
	return toPublicResponses(posts), nil
}

func (s *Service) GetPublic(slug string) (*PostResponse, error) {
	p, err := s.repo.FindPublicBySlug(slug)
	if err != nil {
		return nil, mapErr(err)
	}
	res := toPublic(*p)
	return &res, nil
}

func (s *Service) AdminUsers() ([]AdminUser, error) {
	users, err := s.repo.UsersWithCounts()
	for i := range users {
		users[i].TotalArticles = users[i].TotalPosts
		users[i].TotalSelesai = users[i].TotalPublished
	}
	return users, err
}

func (s *Service) AdminPosts() ([]PostResponse, error) {
	posts, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	return toResponses(posts), nil
}
