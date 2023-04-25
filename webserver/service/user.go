package service

import (
	"fmt"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (s *Service) GetUserInfo(id uint64) (storage.User, error) {
	var user storage.User
	if err := s.db.Where("id = ?", id).Find(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, utils.NewError(fmt.Errorf("user not found"), utils.ErrorNotFound)
		}
		return user, err
	}
	return user, nil
}

func (s *Service) UpdateUserInfo(id uint64, userInfo portal.UpdateUserRequest) (storage.User, error) {
	var user storage.User
	if err := s.db.Where("id = ?", id).Find(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, utils.NewError(fmt.Errorf("user not found"), utils.ErrorNotFound)
		}
		return user, err
	}

	// check email duplicate
	if !utils.IsEmpty(userInfo.Email) {
		var oldUser storage.User
		var err = s.db.Where("email", user.Email).Not("id", user.Id).First(&oldUser).Error
		if err == nil {
			return user, fmt.Errorf("the email is already taken")
		}
	}

	utils.SetValue(&user.DisplayName, userInfo.DisplayName)
	utils.SetValue(&user.Email, userInfo.Email)
	utils.SetValue(&user.Otp, userInfo.Otp)
	utils.SetValue(&user.HourlyLaborRate, userInfo.HourlyLaborRate)
	user.PaymentSettings = userInfo.PaymentSettings

	if !utils.IsEmpty(userInfo.Password) {
		hash, err := bcrypt.GenerateFromPassword([]byte(userInfo.Password), bcrypt.DefaultCost)
		if err != nil {
			return user, err
		}
		user.PasswordHash = string(hash)
	}

	if err := s.db.Save(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}
