package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user disabled")
	ErrUserPending        = errors.New("user pending")
)

type userStore interface {
	Create(ctx context.Context, name, email, passwordHash string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type AuthService struct {
	users     userStore
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewAuthService(users userStore, jwtSecret string) *AuthService {
	return &AuthService{
		users:     users,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  24 * time.Hour,
	}
}

type AuthResult struct {
	User  *model.User
	Token string
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (*AuthResult, error) {
	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailTaken
	}

	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("check email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, name, email, string(hash))

	if err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}

	token, err := s.issueToken(user)

	if err != nil {
		return nil, err
	}

	user.Password = ""
	return &AuthResult{
		User:  user,
		Token: token,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}

	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	if user.Status == model.UserDisabled {
		return nil, ErrUserDisabled
	}

	if user.Status == model.UserPending {
		return nil, ErrUserPending
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.issueToken(user)
	if err != nil {
		return nil, err
	}

	user.Password = ""
	return &AuthResult{
		User:  user,
		Token: token,
	}, nil

}

func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}

	if err != nil {
		return nil, err
	}

	user.Password = ""
	return user, nil
}

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *AuthService) issueToken(user *model.User) (string, error) {
	c := claims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "mini-evvy",
			Audience:  jwt.ClaimStrings{"api"},
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := t.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}
