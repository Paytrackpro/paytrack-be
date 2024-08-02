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

func (s *Service) GetUserTimer(timerId uint64) (storage.UserTimer, error) {
	var userTimer storage.UserTimer
	if err := s.db.Where("id = ?", timerId).Find(&userTimer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return userTimer, utils.NewError(fmt.Errorf("user timer not found"), utils.ErrorNotFound)
		}
		log.Error("GetUserTimer:get user timer info fail with error: ", err)
		return userTimer, err
	}
	return userTimer, nil
}

func (s *Service) GetAdminIds() ([]uint64, error) {
	adminIds := make([]uint64, 0)
	query := "SELECT id FROM users WHERE role = 1 AND locked = false"
	if err := s.db.Raw(query).Scan(&adminIds).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return adminIds, nil
		}
		return nil, err
	}
	return adminIds, nil
}

func (s *Service) GetWorkingUserList() (map[uint64]bool, error) {
	type UserPausing struct {
		UserId  uint64 `json:"userId"`
		Pausing bool   `json:"pausing"`
	}
	userPausingList := make([]UserPausing, 0)
	resMap := make(map[uint64]bool)
	query := "SELECT user_id,pausing FROM user_timer WHERE fininshed = false"
	if err := s.db.Raw(query).Scan(&userPausingList).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return resMap, nil
		}
		return nil, err
	}
	for _, userPausing := range userPausingList {
		resMap[userPausing.UserId] = userPausing.Pausing
	}
	return resMap, nil
}

func (s *Service) GetLogTimeList(userId uint64, request storage.AdminReportFilter) ([]storage.UserTimer, error) {
	timerList := make([]storage.UserTimer, 0)
	page := request.Page
	if page != 0 {
		page -= 1
	}
	query := fmt.Sprintf(`SELECT * FROM user_timer WHERE user_id = %d AND (start AT TIME ZONE 'UTC') < '%s' AND (start AT TIME ZONE 'UTC') > '%s' ORDER BY start DESC LIMIT %d OFFSET %d`,
		userId, utils.TimeToStringWithoutTimeZone(request.EndDate), utils.TimeToStringWithoutTimeZone(request.StartDate), request.Size, request.Size*page)
	if err := s.db.Raw(query).Scan(&timerList).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return timerList, nil
		}
		return nil, err
	}
	return timerList, nil
}

func (s *Service) CountLogTimer(userId uint64, request storage.AdminReportFilter) (int64, error) {
	var count int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM user_timer WHERE user_id = %d AND (start AT TIME ZONE 'UTC') < '%s' AND (start AT TIME ZONE 'UTC') > '%s'`,
		userId, utils.TimeToStringWithoutTimeZone(request.EndDate), utils.TimeToStringWithoutTimeZone(request.StartDate))

	if err := s.db.Raw(countQuery).Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) GetRunningTimer(userId uint64) (*storage.UserTimer, error) {
	var userTimer storage.UserTimer
	if err := s.db.Where("user_id = ? AND fininshed = ?", userId, false).First(&userTimer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &userTimer, nil
}

func (s *Service) GetStoreIdAndApikeyFromUser(userId uint64) (string, string, error) {
	var user storage.User
	if err := s.db.Where("id = ? AND use_btc_pay AND coalesce(store_id, '') <> '' AND coalesce(btc_key, '') <> ''", userId).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", utils.NewError(fmt.Errorf("user not found"), utils.ErrorNotFound)
		}
		log.Error("GetStoreIdFromUser:get user fail with error: ", err)
		return "", "", err
	}
	return user.StoreId, user.BtcKey, nil
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
	utils.SetValue(&user.ShowDateOnInvoiceLine, userInfo.ShowDateOnInvoiceLine)
	utils.SetValue(&user.ShowDraftForRecipient, userInfo.ShowDraftForRecipient)
	utils.SetValue(&user.HidePaid, userInfo.HidePaid)
	utils.SetValue(&user.UseBTCPay, userInfo.UseBTCPay)
	if userInfo.UseBTCPay {
		utils.SetValue(&user.BtcKey, userInfo.BtcKey)
		utils.SetValue(&user.StoreId, userInfo.StoreId)
	}
	if isAdmin {
		user.Role = userInfo.Role
	}
	user.PaymentSettings = userInfo.PaymentSettings
	uName := ""
	oldDisplayName := user.DisplayName
	oldShopName := user.ShopName
	oldUserName := user.UserName

	if isAdmin {
		// if user.UserName was changed, check duplicate username, sync with payment data
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
	user.DisplayName = userInfo.DisplayName
	user.ShopName = userInfo.ShopName

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

	if err := s.SyncPaymentUser(tx, int(user.Id), userInfo.DisplayName, oldDisplayName, uName); err != nil {
		log.Error("UpdateUserInfo: Sync payment user fail with error: ", err)
		tx.Rollback()
		return user, err
	}

	var shopName = ""
	if strings.Compare(userInfo.ShopName, oldShopName) != 0 {
		shopName = GetShopName(userInfo.UserName, userInfo.DisplayName, userInfo.ShopName)
	}
	var ownerName = ""
	if strings.Compare(userInfo.DisplayName, oldDisplayName) != 0 || (utils.IsEmpty(userInfo.DisplayName) && strings.Compare(userInfo.UserName, oldUserName) != 0) {
		ownerName = GetDisplayName(userInfo.UserName, userInfo.DisplayName)
	}
	if err := s.SyncShopUserInfo(tx, int(user.Id), shopName, ownerName); err != nil {
		log.Error("UpdateUserInfo: Sync payment user fail with error: ", err)
		tx.Rollback()
		return user, err
	}

	tx.Commit()
	return user, nil
}

func GetShopName(uName string, displayName string, shopName string) string {
	if utils.IsEmpty(shopName) {
		return GetDisplayName(uName, displayName)
	}
	return shopName
}

func GetDisplayName(uName string, displayName string) string {
	if utils.IsEmpty(displayName) {
		return uName
	}
	return displayName
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

	if err := s.SyncPaymentUser(tx, int(user.Id), uDisplayName, "", ""); err != nil {
		log.Error("UpdateUserInfo: Sync payment user fail with error: ", err)
		tx.Rollback()
		return user, err
	}

	tx.Commit()
	return user, nil
}
