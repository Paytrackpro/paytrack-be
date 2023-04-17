package service

import (
	"fmt"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"gorm.io/gorm"
)

func (s *Service) GetBulkPaymentBTC(userId uint64, page, pageSize int) ([]storage.Payment, int64, error) {
	if page == 1 {
		page = page - 1
	}
	var count int64
	payments := make([]storage.Payment, 0)
	offset := page * pageSize

	build := s.db.Table("payments").
		Select("payments.*, sender.user_name as sender_name, receiver.user_name as receiver_name").
		Joins("JOIN users as sender ON payments.sender_id = sender.id").
		Joins("JOIN users as receiver ON payments.receiver_id = receiver.id").
		Where("payments.payment_method = ? AND payments.status = ? AND payments.receiver_id = ?", utils.PaymentTypeBTC, storage.PaymentStatusConfirmed, userId).
		Scan(&payments)

	buildCount := s.db.Model(&storage.Payment{}).Where("payment_method = ? AND status = ? AND receiver_id = ?", utils.PaymentTypeBTC, storage.PaymentStatusConfirmed, userId)
	if err := buildCount.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	build = build.Limit(pageSize).Offset(offset)
	if err := build.Debug().Find(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, 0, nil
		}
		return nil, 0, err
	}

	return payments, count, nil
}

func (s *Service) CreatePayment(userId uint64, userName string, request portal.PaymentRequest) (*storage.Payment, error) {
	var reciver storage.User
	payment := storage.Payment{
		SenderId:        userId,
		SenderName:      userName,
		Description:     request.Description,
		Details:         request.Details,
		Status:          request.Status,
		HourlyRate:      request.HourlyRate,
		PaymentSettings: request.PaymentSettings,
	}

	// payment is internal
	if request.ContactMethod == storage.PaymentTypeInternal {
		if err := s.db.Where("id = ?", request.ReceiverId).First(&reciver).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, utils.NewError(fmt.Errorf("receiver not found"), utils.ErrorBadRequest)
			}
			return nil, err
		}
		payment.ReceiverId = request.ReceiverId
		payment.ReceiverName = reciver.UserName
	} else {
		// payment is external
		payment.ExternalEmail = request.ExternalEmail
	}

	if len(request.Details) > 0 {
		amount, err := calculateAmount(request)
		if err != nil {
			return nil, utils.NewError(err, utils.ErrorBadRequest)
		}
		payment.Amount = amount
	} else {
		payment.Amount = request.Amount
	}

	if payment.Status == storage.PaymentStatusSent {
		approverSettings, err := s.GetApproverForPayment(userId, payment.ReceiverId)
		if err != nil {
			return nil, err
		}

		if len(approverSettings) > 0 {
			approvers := storage.Approvers{}
			for _, approver := range approverSettings {
				approvers = append(approvers, storage.Approver{
					ApproverId:   approver.Id,
					ApproverName: approver.ApproverName,
					IsApproved:   false,
				})
			}
			payment.Approvers = approvers
		}
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (s *Service) UpdatePayment(id, userId uint64, request portal.PaymentRequest) (*storage.Payment, error) {
	var payment storage.Payment
	if err := s.db.First("id = ?", id, &payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(fmt.Errorf("payment not found with id %d", id), utils.ErrorNotFound)
		}
		return nil, err
	}

	if userId == 0 || request.ReceiverId == userId {
		// receiver or external update
		// allow recipient update status to sent or confirmed
		if payment.Status == storage.PaymentStatusSent || payment.Status == storage.PaymentStatusConfirmed {
			payment.Status = request.Status
		}
		payment.TxId = request.TxId
	} else {
		// sender update
		payment.Description = request.Description
		payment.Details = request.Details
		payment.HourlyRate = request.HourlyRate
		payment.PaymentSettings = request.PaymentSettings
		if len(request.Details) > 0 {
			amount, err := calculateAmount(request)
			if err != nil {
				return nil, utils.NewError(err, utils.ErrorBadRequest)
			}
			payment.Amount = amount
		} else {
			payment.Amount = request.Amount
		}

		// use for sender update status from save as draft to sent
		if payment.Status == storage.PaymentStatusSent && payment.Status != request.Status {
			approverSettings, err := s.GetApproverForPayment(userId, payment.ReceiverId)
			if err != nil {
				return nil, err
			}

			if len(approverSettings) > 0 {
				approvers := storage.Approvers{}
				for _, approver := range approverSettings {
					approvers = append(approvers, storage.Approver{
						ApproverId:   approver.Id,
						ApproverName: approver.ApproverName,
						IsApproved:   false,
					})
				}
				payment.Approvers = approvers
			}
		}
		payment.Status = request.Status
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (s *Service) BulkPaidBTC(userId uint64, txId string, paymentIds []int) error {
	payments := make([]*storage.Payment, 0)
	if err := s.db.Where("id IN ?", paymentIds).Find(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	// validate payment
	for _, paym := range payments {
		if paym.Status != storage.PaymentStatusConfirmed {
			return fmt.Errorf("all payments need to be ready for payment")
		}

		if paym.ReceiverId != userId {
			return fmt.Errorf("all payments must be yours")
		}

		if paym.PaymentMethod != utils.PaymentTypeBTC {
			return fmt.Errorf("all payments needs the payment method to be BTC")
		}
		paym.TxId = txId
		paym.PaidAt = time.Now()
		paym.Status = storage.PaymentStatusPaid
	}

	if err := s.db.Save(&payments).Error; err != nil {
		fmt.Println("ERROR: ", err)
		return &utils.InternalError
	}
	return nil
}

func calculateAmount(request portal.PaymentRequest) (float64, error) {
	var amount float64
	for i, detail := range request.Details {
		if detail.Quantity > 0 {
			var price = request.HourlyRate
			if detail.Price > 0 {
				price = detail.Price
			}
			cost := detail.Quantity * price
			if cost != detail.Cost {
				return 0, fmt.Errorf("payment detail amount is incorrect at line %d", i+1)
			}
			if detail.Cost <= 0 {
				return 0, fmt.Errorf("payment detail cost must be greater than 0 at line %d", i+1)
			}
		}
		amount += detail.Cost
	}
	return amount, nil
}
