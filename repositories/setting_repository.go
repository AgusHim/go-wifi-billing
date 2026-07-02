package repositories

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingRepository interface {
	GetAll() ([]models.Setting, error)
	GetByKey(key string) (*models.Setting, error)
	UpdateOrCreate(key string, value string) (*models.Setting, error)
}

type settingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db}
}

func (r *settingRepository) GetAll() ([]models.Setting, error) {
	var settings []models.Setting
	err := r.db.Find(&settings).Error
	return settings, err
}

func (r *settingRepository) GetByKey(key string) (*models.Setting, error) {
	var setting models.Setting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *settingRepository) UpdateOrCreate(key string, value string) (*models.Setting, error) {
	var setting models.Setting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			setting = models.Setting{
				ID:    uuid.New(),
				Key:   key,
				Value: value,
			}
			if err := r.db.Create(&setting).Error; err != nil {
				return nil, err
			}
			return &setting, nil
		}
		return nil, err
	}

	setting.Value = value
	if err := r.db.Omit(clause.Associations).Save(&setting).Error; err != nil {
		return nil, err
	}

	return &setting, nil
}
