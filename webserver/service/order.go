package service

import (
	"fmt"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
)

func (s *Service) CreateOrders(userId uint64, request portal.OrderForm) (*[]storage.Order, error) {
	tx := s.db.Begin()
	var orders []storage.Order
	for _, orderData := range request.OrderData {
		var order = storage.Order{
			UserId:          userId,
			OwnerId:         orderData.OwnerId,
			PhoneNumber:     orderData.PhoneNumber,
			Address:         orderData.Address,
			Memo:            orderData.Memo,
			ProductPayments: orderData.ProductPayments,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := s.db.Create(&order).Error; err != nil {
			return nil, err
		}
		//Delete Product from cart
		for _, productPayment := range orderData.ProductPayments {
			fmt.Println(productPayment.Amount)
			s.DeleteCart(userId, productPayment.ProductId)
		}
		orders = append(orders, order)
	}
	tx.Commit()
	return &orders, nil
}
