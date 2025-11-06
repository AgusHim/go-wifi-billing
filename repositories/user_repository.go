package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Agushim/go_wifi_billing/models"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByEmail(email string) (*models.User, error)
	GetByID(id uuid.UUID) (*models.User, error)
	GetAll() ([]models.User, error)
	Update(user *models.User) error
	Delete(id uuid.UUID) error
	CheckIsRegistered(email string, phone string) (*models.User, error)
	FindIncludeDeleted(user *models.User, email, phone string) error
	Restore(userID uuid.UUID) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetByID(id uuid.UUID) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, "id = ? AND deleted_at IS NULL", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.db.Find(&users).Error
	return users, err
}

func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}

func (r *userRepository) CheckIsRegistered(email string, phone string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("deleted_at IS NULL").Where("email = ? OR phone = ?", email, phone).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindIncludeDeleted(user *models.User, email, phone string) error {
	return r.db.Unscoped().Where("email = ? OR phone = ?", email, phone).First(user).Error
}

func (r *userRepository) Restore(userID uuid.UUID) error {
	return r.db.Unscoped().Model(&models.User{}).Where("id = ?", userID).Update("deleted_at", nil).Error
}
