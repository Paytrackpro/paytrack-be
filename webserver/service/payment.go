package service

import (
	"fmt"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
)

func (s *Service) GetBulkPaymentBTC(userId uint64, page, pageSize int) ([]storage.Payment, int64, error) {
	if page == 1 {
		page = page - 1
	}
	var count int64
	payments := make([]storage.Payment, 0)
	offset := page * pageSize

	build := s.db.Where("payment_method = ? AND status = ? AND receiver_id = ?", payment.PaymentTypeBTC, storage.PaymentStatusConfirmed, userId)
	buildCount := s.db.Model(&storage.Payment{}).Where("payment_method = ? AND status = ? AND receiver_id = ?", payment.PaymentTypeBTC, storage.PaymentStatusConfirmed, userId)
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
			return fmt.Errorf("all payments need ready for payment")
		}

		if paym.ReceiverId != userId {
			return fmt.Errorf("all payments must be your")
		}

		if paym.PaymentMethod != payment.PaymentTypeBTC {
			return fmt.Errorf("all payments need payment method is BTC")
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
