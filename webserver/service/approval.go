package service

import (
	"fmt"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) ApproverPaymentRequest(id, status, userId uint64, userName string) (*storage.Payment, error) {
	var payment storage.Payment
	if err := s.db.First(&payment, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, err
	}

	// Get all approver for payment
	approvers := make([]storage.ApproverSettings, 0)
	if err := s.db.Where("send_user_id = ? AND recipient_id = ?", payment.SenderId, payment.ReceiverId).Find(&approvers).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("you do not have permission to approve this invoice")
		}
		return nil, err
	}

	// Check current user is approver
	isApprover := false
	for _, approver := range approvers {
		if approver.ApproverId == userId {
			isApprover = true
		}
	}

	if !isApprover {
		return nil, fmt.Errorf("you do not have permission to approve this invoice")
	}

	if len(payment.Approvers) == 0 {
		payment.Approvers = append(payment.Approvers, storage.Approver{
			ApproverId:   userId,
			ApproverName: userName,
			Status:       status,
		})
	} else {
		isNewApprover := true
		for i, appro := range payment.Approvers {
			//check and change status of user approver
			if appro.ApproverId == userId {
				isNewApprover = false
				payment.Approvers[i].Status = status
				payment.Approvers[i].ApproverName = userName
			}
		}
		if isNewApprover {
			payment.Approvers = append(payment.Approvers, storage.Approver{
				ApproverId:   userId,
				ApproverName: userName,
				Status:       status,
			})
		}
	}

	if len(approvers) <= len(payment.Approvers) {
		payment.Status = storage.PaymentStatusApproved
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}

	payment.Status = storage.PaymentStatus(status)
	return &payment, nil
}

func (s *Service) GetSettingOfApprover(id uint64) ([]storage.ApproverSettings, error) {
	approvers := make([]storage.ApproverSettings, 0)
	if err := s.db.Where("approver_id = ?", id).Find(&approvers).Error; err != nil {
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
