package repository

import (
	"errors"
	"transcribe/config"
	"transcribe/internal/domain"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Create(user *domain.RegisterRequest) (*domain.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)

	if err != nil {
		return nil, err
	}

	newUser := domain.User{
		Name:     user.Name,
		Email:    user.Email,
		Password: string(hashedPassword),
	}

	result := config.DB.Create(&newUser)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return nil, errors.New("email already exists")
		}
		return nil, result.Error
	}

	return &newUser, nil
}

func (r *UserRepository) GetAll(page, pageSize int) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	offset := (page - 1) * pageSize

	config.DB.Model(&domain.User{}).Count(&total)

	result := config.DB.Offset(offset).Limit(pageSize).Find(&users)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return users, total, nil
}

func (r *UserRepository) Update(id uint, updates map[string]interface{}) error {
	result := config.DB.Model(&domain.User{}).Where("id = ?", id).Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *UserRepository) Delete(id uint) error {
	result := config.DB.Delete(&domain.User{}, id)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User

	result := config.DB.Where("email = ?", email).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return &user, nil
}

func (r *UserRepository) FindByID(id uint) (*domain.User, error) {
	var user domain.User

	result := config.DB.First(&user, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return &user, nil
}

func (r *UserRepository) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
