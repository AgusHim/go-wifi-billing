package services

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
)

var jwtSecret []byte

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("warn: .env not loaded: %v", err)
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		if strings.HasSuffix(os.Args[0], ".test") {
			secret = "test-jwt-secret"
		} else {
			panic("JWT_SECRET environment variable is required")
		}
	}
	jwtSecret = []byte(secret)
}

type UserService interface {
	Register(input dto.RegisterDTO) (*models.User, error)
	Create(input dto.CreateUserDTO) (*models.User, error)
	Login(input dto.LoginDTO) (string, *models.User, error)
	GetAll(page int, limit int, roles []string, search string) ([]models.User, int64, error)
	GetByID(id string) (*models.User, error)
	UpdateProfile(id string, input dto.UpdateProfileDTO) (*models.User, error)
	Update(id string, input dto.UpdateUserDTO) (*models.User, error)
	Delete(id string) error
	CheckIsRegistered(email string, phone string) (*models.User, error)
	CheckIsRegisteredIncludeDeleted(email, phone string) (*models.User, error)
	Restore(userID uuid.UUID) error
}

type userService struct {
	repo repositories.UserRepository
}

func NewUserService(r repositories.UserRepository) UserService {
	return &userService{repo: r}
}

func (s *userService) Register(input dto.RegisterDTO) (*models.User, error) {
	return s.createUser(input.Name, input.Email, input.Phone, input.Password, "user", "customer")
}

func (s *userService) Create(input dto.CreateUserDTO) (*models.User, error) {
	legacyRole, canonicalRole, err := assignableRole(input.Role)
	if err != nil {
		return nil, err
	}
	return s.createUser(input.Name, input.Email, input.Phone, input.Password, legacyRole, canonicalRole)
}

func (s *userService) createUser(name, email, phone, password, legacyRole, canonicalRole string) (*models.User, error) {
	existing, _ := s.repo.GetByEmail(email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	if strings.TrimSpace(name) == "" || strings.TrimSpace(email) == "" || len(password) < 6 {
		return nil, errors.New("name, email, and password with at least 6 characters are required")
	}
	role, err := s.repo.GetRoleByKey(canonicalRole)
	if err != nil {
		return nil, fmt.Errorf("role %s is not available: %w", canonicalRole, err)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:     strings.TrimSpace(name),
		Email:    strings.TrimSpace(email),
		Phone:    strings.TrimSpace(phone),
		Password: string(hashed),
		Role:     legacyRole,
		RoleID:   &role.ID,
		IsActive: true,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) Login(input dto.LoginDTO) (string, *models.User, error) {
	user, err := s.repo.GetByEmail(input.Email)
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		return "", nil, errors.New("invalid email or password")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return t, user, nil
}

func (s *userService) GetAll(page int, limit int, roles []string, search string) ([]models.User, int64, error) {
	return s.repo.GetAll(page, limit, roles, search)
}

func (s *userService) GetByID(id string) (*models.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(uid)
}

func (s *userService) UpdateProfile(id string, input dto.UpdateProfileDTO) (*models.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	user, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	applyProfileUpdate(user, input)
	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) Update(id string, input dto.UpdateUserDTO) (*models.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	user, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	applyProfileUpdate(user, dto.UpdateProfileDTO{Name: input.Name, Email: input.Email, Phone: input.Phone})
	if input.Password != nil && strings.TrimSpace(*input.Password) != "" {
		if len(*input.Password) < 6 {
			return nil, errors.New("password must be at least 6 characters")
		}
		hashed, hashErr := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, hashErr
		}
		user.Password = string(hashed)
	}
	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func applyProfileUpdate(user *models.User, input dto.UpdateProfileDTO) {
	if input.Name != nil && strings.TrimSpace(*input.Name) != "" {
		user.Name = strings.TrimSpace(*input.Name)
	}
	if input.Email != nil && strings.TrimSpace(*input.Email) != "" {
		user.Email = strings.TrimSpace(*input.Email)
	}
	if input.Phone != nil {
		user.Phone = strings.TrimSpace(*input.Phone)
	}
}

func assignableRole(input string) (legacyRole string, canonicalRole string, err error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "admin":
		return "admin", "admin", nil
	case "petugas":
		return "petugas", "petugas", nil
	case "loket":
		return "loket", "loket", nil
	case "teknisi":
		return "teknisi", "teknisi", nil
	case "user", "customer":
		return "user", "customer", nil
	case "owner":
		return "", "", errors.New("owner role can only be assigned through owner-only access control")
	default:
		return "", "", errors.New("invalid role")
	}
}

func (s *userService) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid user id")
	}
	user, err := s.repo.GetByID(uid)
	if err != nil {
		return err
	}
	if user.RoleDefinition != nil && user.RoleDefinition.IsOwner {
		return errors.New("owner cannot be deleted through user administration")
	}
	if canonicalRole, known := models.CanonicalRoleKey(user.Role); known && canonicalRole == "owner" {
		return errors.New("owner cannot be deleted through user administration")
	}
	return s.repo.Delete(uid)
}

func (s *userService) CheckIsRegistered(email string, phone string) (*models.User, error) {
	return s.repo.CheckIsRegistered(email, phone)
}

func (s *userService) CheckIsRegisteredIncludeDeleted(email, phone string) (*models.User, error) {
	var user models.User
	if err := s.repo.FindIncludeDeleted(&user, email, phone); err != nil {
		return nil, err
	}
	if user.ID == uuid.Nil {
		return nil, nil
	}
	return &user, nil
}

func (s *userService) Restore(userID uuid.UUID) error {
	return s.repo.Restore(userID)
}
