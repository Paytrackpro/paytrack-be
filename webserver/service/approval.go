package service

import (
	"fmt"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"gorm.io/gorm"
)

type Config struct {
	Exchange        string `yaml:"exchange"`
	CoimarketcapKey string `yaml:"coimarketcapKey"`
}

type Service struct {
	db              *gorm.DB
	exchange        string
	coinMaketCapKey string
}

func NewService(conf Config, db *gorm.DB) *Service {
	return &Service{
		db:              db,
		exchange:        conf.Exchange,
		coinMaketCapKey: conf.CoimarketcapKey,
	}
}

func (s *Service) ApprovePaymentRequest(id, userId uint64) (*storage.Payment, error) {
	var payment storage.Payment
	if err := s.db.First(&payment, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, err
	}
	for i, approver := range payment.Approvers {
		if approver.ApproverId == userId {
			payment.Approvers[i].IsApproved = true
		}
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

func (s *Service) GetSettingOfApprover(id uint64) ([]storage.ApproverSettings, error) {
	approvers := make([]storage.ApproverSettings, 0)
	if err := s.db.Where("approver_id = ?", id).Find(&approvers).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return approvers, nil
		}
		return nil, err
	}
	return approvers, nil
}

func (s *Service) GetApprovalSetting(sendId, recipientId, approverId uint64) (*storage.ApproverSettings, error) {
	var apst storage.ApproverSettings
	if err := s.db.First(&apst, "approver_id = ? AND recipient_id = ? AND send_user_id = ?", approverId, recipientId, sendId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &apst, nil
}

func (s *Service) GetApproverForPayment(sendId, recipientId uint64) ([]storage.ApproverSettings, error) {
	apst := make([]storage.ApproverSettings, 0)
	if err := s.db.Where("recipient_id = ? AND send_user_id = ?", recipientId, sendId).Find(&apst).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return apst, nil
}

func (s *Service) UpdateApproverSetting(userId uint64, approvers []portal.ApproversSettingRequest) ([]storage.ApproverSettings, error) {
	//Get all user is sender and approvers
	userIds := make([]uint64, 0)
	for _, approver := range approvers {
		userIds = append(userIds, approver.SendUserId)
		userIds = append(userIds, approver.ApproverIds...)
	}

	var users []storage.User
	//get all user on setting
	if err := s.db.Where("id IN ?", userIds).Find(&users).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(fmt.Errorf("sender user or approver user not found"), utils.ErrorBadRequest)
		}
		return nil, err
	}

	userMap := make(map[uint64]storage.User, 0)
	//conver slices user to map
	for _, user := range users {
		userMap[user.Id] = user
	}

	approversMap := make(map[uint64]storage.ApproverSettings, 0)
	settingApprovers := make([]storage.ApproverSettings, 0)
	for _, setting := range approvers {
		for _, v := range setting.ApproverIds {
			app := storage.ApproverSettings{
				ApproverId:   v,
				SendUserId:   setting.SendUserId,
				RecipientId:  userId,
				ApproverName: userMap[v].UserName,
				SendUserName: userMap[setting.SendUserId].UserName,
			}
			approversMap[v] = app
			settingApprovers = append(settingApprovers, app)
		}
	}

	//conver slices user to map
	for _, user := range users {
		userMap[user.Id] = user
	}

	// update all payment pendding
	payments := make([]*storage.Payment, 0)
	if err := s.db.Where("receiver_id = ? AND status = ?", userId, storage.PaymentStatusSent).Find(&payments).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	//update approver for all payment
	for i, payment := range payments {
		approverMap := make(map[uint64]storage.Approver, 0)
		for _, approver := range payment.Approvers {
			approverMap[approver.ApproverId] = approver
		}
		newApprovers := make([]storage.Approver, 0)
		for _, app := range settingApprovers {
			approved := false
			ap, ok := approverMap[app.ApproverId]
			if ok {
				approved = ap.IsApproved
			}
			newApprovers = append(newApprovers, storage.Approver{
				ApproverId:   app.ApproverId,
				ApproverName: app.ApproverName,
				IsApproved:   approved,
			})
		}
		payments[i].Approvers = newApprovers
	}

	// Save to DB
	tx := s.db.Begin()

	// delete all old approver setting
	if err := tx.Where("recipient_id", userId).Delete(storage.ApproverSettings{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// save new data
	if err := tx.Create(&settingApprovers).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// update payment
	if len(payments) > 0 {
		if err := tx.Save(&payments).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	tx.Commit()
	return settingApprovers, nil
}
