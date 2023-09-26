package service

import (
	"fmt"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"gorm.io/gorm"
)

func (s *Service) GetProductInfo(id uint64) (storage.Product, error) {
	var product storage.Product
	if err := s.db.Where("id = ?", id).Find(&product).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return product, utils.NewError(fmt.Errorf("product not found"), utils.ErrorNotFound)
		}
		log.Error("getProductInfo:get product info fail with error: ", err)
		return product, err
	}
	return product, nil
}

func (s *Service) UpdateProduct(id uint64, productInfo portal.UpdateProductRequest) (storage.Product, error) {
	var product storage.Product
	if err := s.db.Where("id = ?", id).First(&product).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return product, utils.NewError(fmt.Errorf("Product not found"), utils.ErrorNotFound)
		}
		log.Error("UpdateProductInfo:get product fail with error: ", err)
		return product, err
	}

	utils.SetValue(&product.ProductCode, productInfo.ProductCode)
	utils.SetValue(&product.ProductName, productInfo.ProductName)
	utils.SetValue(&product.Description, productInfo.Description)
	utils.SetValue(&product.Currency, productInfo.Currency)
	utils.SetValue(&product.Avatar, productInfo.Avatar)
	utils.SetValue(&product.Images, productInfo.Images)
	utils.SetValue(&product.Price, productInfo.Price)
	utils.SetValue(&product.Stock, productInfo.Stock)
	utils.SetValue(&product.Status, productInfo.Status)

	tx := s.db.Begin()

	if err := tx.Save(&product).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateProduct:save product fail with error: ", err)
		return product, err
	}

	tx.Commit()
	return product, nil
}

// func (s *Service) UpdateUserInfos(id uint64, userInfo portal.UpdateUserRequest) (storage.User, error) {
// 	var user storage.User
// 	if err := s.db.Where("id = ?", id).First(&user).Error; err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			return user, utils.NewError(fmt.Errorf("user not found"), utils.ErrorNotFound)
// 		}
// 		log.Error("UpdateUserInfo:get user fail with error: ", err)
// 		return user, err
// 	}

// 	// check email duplicate
// 	if !utils.IsEmpty(userInfo.Email) && user.Email != userInfo.Email {
// 		var oldUser storage.User
// 		var err = s.db.Debug().Where("email", userInfo.Email).Not("id", user.Id).First(&oldUser).Error
// 		if err == nil {
// 			return user, fmt.Errorf("the email is already taken")
// 		} else if err != gorm.ErrRecordNotFound {
// 			log.Error("UpdateUserInfo:check email duplicate fail with error: ", err)
// 			return user, err
// 		}
// 	}

// 	utils.SetValue(&user.Email, userInfo.Email)
// 	utils.SetValue(&user.Otp, userInfo.Otp)
// 	utils.SetValue(&user.HourlyLaborRate, userInfo.HourlyLaborRate)
// 	user.PaymentSettings = userInfo.PaymentSettings

// 	uDisplayName := ""
// 	// if user.DisplayName was changed, sync with payment data
// 	if len(userInfo.DisplayName) > 0 && strings.Compare(userInfo.DisplayName, user.DisplayName) != 0 {
// 		uDisplayName = userInfo.DisplayName
// 	}

// 	utils.SetValue(&user.DisplayName, userInfo.DisplayName)

// 	if !utils.IsEmpty(userInfo.Password) {
// 		hash, err := bcrypt.GenerateFromPassword([]byte(userInfo.Password), bcrypt.DefaultCost)
// 		if err != nil {
// 			return user, err
// 		}
// 		user.PasswordHash = string(hash)
// 	}

// 	tx := s.db.Begin()

// 	if err := tx.Save(&user).Error; err != nil {
// 		log.Error("UpdateUserInfo: save user fail with error: ", err)
// 		tx.Rollback()
// 		return user, err
// 	}

// 	if err := s.SyncPaymentUser(tx, int(user.Id), uDisplayName, ""); err != nil {
// 		log.Error("UpdateUserInfo: Sync payment user fail with error: ", err)
// 		tx.Rollback()
// 		return user, err
// 	}

// 	tx.Commit()
// 	return user, nil
// }
