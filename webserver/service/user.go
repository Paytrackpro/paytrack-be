package service

import (
	"fmt"
	"strings"

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
		log.Error("GetUserInfo:get user info fail with error: ", err)
		return user, err
	}
	return user, nil
}

func (s *Service) UpdateUserInfo(id uint64, userInfo portal.UpdateUserRequest, isAdmin bool) (storage.User, error) {
	var user storage.User
	if err := s.db.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, utils.NewError(fmt.Errorf("user not found"), utils.ErrorNotFound)
		}
		log.Error("UpdateUserInfo:get user fail with error: ", err)
		return user, err
	}

	// check email duplicate
	if !utils.IsEmpty(userInfo.Email) && user.Email != userInfo.Email {
		var oldUser storage.User
		var err = s.db.Where("email", userInfo.Email).Not("id", user.Id).First(&oldUser).Error
		if err == nil {
			return user, fmt.Errorf("the email is already taken")
		} else if err != gorm.ErrRecordNotFound {
			log.Error("UpdateUserInfo:check email duplicate fail with error: ", err)
			return user, err
		}
	}

	utils.SetValue(&user.Email, userInfo.Email)
	utils.SetValue(&user.HourlyLaborRate, userInfo.HourlyLaborRate)
	user.PaymentSettings = userInfo.PaymentSettings

	uName := ""
	uDisplayName := ""
	// if user.DisplayName was changed, sync with payment data
	if len(userInfo.DisplayName) > 0 && strings.Compare(userInfo.DisplayName, user.DisplayName) != 0 {
		uDisplayName = userInfo.DisplayName
	}

	if isAdmin {
		// if user.UserName was changed, checkduplicate username, sync with payment data
		if len(userInfo.UserName) > 0 && strings.Compare(userInfo.UserName, user.UserName) != 0 {
			var oldUser storage.User
			var err = s.db.Where("user_name", userInfo.UserName).Not("id", user.Id).First(&oldUser).Error
			if err == nil {
				return user, fmt.Errorf("the username is already taken")
			} else if err != gorm.ErrRecordNotFound {
				log.Error("UpdateUserInfo:check username duplicate fail with error: ", err)
				return user, err
			}
			uName = userInfo.UserName
		}

		//if admin update, otp flag is reset OTP.
		if userInfo.Otp {
			user.Otp = false
			user.Secret = ""
		}
		utils.SetValue(&user.Locked, userInfo.Locked)
	} else {
		utils.SetValue(&user.Otp, userInfo.Otp)
	}

	utils.SetValue(&user.UserName, userInfo.UserName)
	utils.SetValue(&user.DisplayName, userInfo.DisplayName)

	if !utils.IsEmpty(userInfo.Password) {
		hash, err := bcrypt.GenerateFromPassword([]byte(userInfo.Password), bcrypt.DefaultCost)
		if err != nil {
			return user, err
		}
		user.PasswordHash = string(hash)
	}

	tx := s.db.Begin()

	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateUserInfo:save user fail with error: ", err)
		return user, err
	}

	if err := s.SyncPaymentUser(tx, int(user.Id), uDisplayName, uName); err != nil {
		log.Error("UpdateUserInfo: Sync payment user fail with error: ", err)
		tx.Rollback()
		return user, err
	}

	tx.Commit()
	return user, nil
}

func (s *Service) UpdateUserInfos(id uint64, userInfo portal.UpdateUserRequest) (storage.User, error) {
	var user storage.User
	if err := s.db.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, utils.NewError(fmt.Errorf("user not found"), utils.ErrorNotFound)
		}
		log.Error("UpdateUserInfo:get user fail with error: ", err)
		return user, err
	}

	// check email duplicate
	if !utils.IsEmpty(userInfo.Email) && user.Email != userInfo.Email {
		var oldUser storage.User
		var err = s.db.Debug().Where("email", userInfo.Email).Not("id", user.Id).First(&oldUser).Error
		if err == nil {
			return user, fmt.Errorf("the email is already taken")
		} else if err != gorm.ErrRecordNotFound {
			log.Error("UpdateUserInfo:check email duplicate fail with error: ", err)
			return user, err
		}
	}

	utils.SetValue(&user.Email, userInfo.Email)
	utils.SetValue(&user.Otp, userInfo.Otp)
	utils.SetValue(&user.HourlyLaborRate, userInfo.HourlyLaborRate)
	user.PaymentSettings = userInfo.PaymentSettings

	uDisplayName := ""
	// if user.DisplayName was changed, sync with payment data
	if len(userInfo.DisplayName) > 0 && strings.Compare(userInfo.DisplayName, user.DisplayName) != 0 {
		uDisplayName = userInfo.DisplayName
	}

	utils.SetValue(&user.DisplayName, userInfo.DisplayName)

	if !utils.IsEmpty(userInfo.Password) {
		hash, err := bcrypt.GenerateFromPassword([]byte(userInfo.Password), bcrypt.DefaultCost)
		if err != nil {
			return user, err
		}
		user.PasswordHash = string(hash)
	}

	tx := s.db.Begin()

	if err := tx.Save(&user).Error; err != nil {
		log.Error("UpdateUserInfo: save user fail with error: ", err)
		tx.Rollback()
		return user, err
	}

	if err := s.SyncPaymentUser(tx, int(user.Id), uDisplayName, ""); err != nil {
		log.Error("UpdateUserInfo: Sync payment user fail with error: ", err)
		tx.Rollback()
		return user, err
	}

	tx.Commit()
	return user, nil
}
