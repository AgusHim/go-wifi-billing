package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
)

type SettingService interface {
	GetAll() ([]models.Setting, error)
	GetByKey(key string) (*models.Setting, error)
	UpdateOrCreate(key string, value string) (*models.Setting, error)
}

type settingService struct {
	repo repositories.SettingRepository
}

func NewSettingService(repo repositories.SettingRepository) SettingService {
	return &settingService{repo}
}

func (s *settingService) GetAll() ([]models.Setting, error) {
	return s.repo.GetAll()
}

func (s *settingService) GetByKey(key string) (*models.Setting, error) {
	return s.repo.GetByKey(key)
}

func (s *settingService) UpdateOrCreate(key string, value string) (*models.Setting, error) {
	return s.repo.UpdateOrCreate(key, value)
}
