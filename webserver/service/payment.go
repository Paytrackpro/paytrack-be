package service

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/storage"
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
