package services

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type UserService interface {
	Register(input dto.RegisterDTO) (*models.User, error)
	Login(input dto.LoginDTO) (string, *models.User, error)
	GetAll(role string) ([]models.User, error)
	GetByID(id string) (*models.User, error)
	Update(id string, input *models.User) (*models.User, error)
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
	existing, _ := s.repo.GetByEmail(input.Email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:     input.Name,
		Email:    input.Email,
		Phone:    input.Phone,
		Password: string(hashed),
		Role:     input.Role,
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

func (s *userService) GetAll(role string) ([]models.User, error) {
	return s.repo.GetAll(role)
}

func (s *userService) GetByID(id string) (*models.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(uid)
}

func (s *userService) Update(id string, input *models.User) (*models.User, error) {
	uid, _ := uuid.Parse(id)
	user, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	user.Name = input.Name
	user.Phone = input.Phone
	user.Email = input.Email
	user.Role = input.Role

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) Delete(id string) error {
	uid, _ := uuid.Parse(id)
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
