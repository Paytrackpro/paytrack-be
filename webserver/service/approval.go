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

	// Check current user is approver
	var approver storage.ApproverSettings
	if err := s.db.First(&approver, "approver_id = ? AND send_user_id = ? AND recipient_id = ?", userId, payment.SenderId, payment.ReceiverId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("you do not have permission to approve this invoice")
		}
		return nil, err
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

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}

	payment.Status = storage.PaymentStatus(status)

	return &payment, nil
}

func (s *Service) GetPaymentOfApprover(id uint64) ([]uint64, error) {
	paymentIds := make([]uint64, 0)
	err := s.db.Model(&storage.Payment{}).Select("payments.id").
		Joins("left join approver_settings on payments.receiver_id = approver_settings.recipient_id AND payments.sender_id = approver_settings.send_user_id").
		Where("approver_id = ?", id).Find(&paymentIds).Error
	if err != nil {
		return nil, err
	}
	return paymentIds, nil
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
