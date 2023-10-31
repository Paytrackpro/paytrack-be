package service

import (
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
)

func (s *Service) CreateOrders(userId uint64, userName string, request portal.OrderForm) (*[]storage.Order, error) {
	tx := s.db.Begin()
	var orders []storage.Order
	for _, orderData := range request.OrderData {
		var order = storage.Order{
			UserId:          userId,
			UserName:        userName,
			OwnerId:         orderData.OwnerId,
			OwnerName:       orderData.OwnerName,
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
			s.DeleteCart(userId, productPayment.ProductId)
		}
		orders = append(orders, order)
	}
	tx.Commit()
	return &orders, nil
}

func (s *Service) GetOrderDetail(orderId uint64) (portal.OrderDisplayData, error) {
	var order storage.Order
	var orderDipslay portal.OrderDisplayData
	if err := s.db.Where("order_id = ?", orderId).Find(&order).Error; err != nil {
		log.Error("getOrderDetail:get order info fail with error: ", err)
		return orderDipslay, err
	}
	return convertToOrderDisplay(order), nil
}

func (s *Service) GetOrderManagement(userId uint64) ([]portal.OrderDisplayData, error) {
	var orders []storage.Order
	if err := s.db.Where("owner_id = ?", userId).Order("updated_at").Find(&orders).Error; err != nil {
		log.Error("getOrderManagement:get orders info fail with error: ", err)
		return nil, err
	}
	var orderDisplay []portal.OrderDisplayData
	for _, order := range orders {
		var tmpOrderDisplay = convertToOrderDisplay(order)
		orderDisplay = append(orderDisplay, tmpOrderDisplay)
	}
	return orderDisplay, nil
}

func (s *Service) GetMyOrders(userId uint64) ([]portal.OrderDisplayData, error) {
	var orders []storage.Order
	if err := s.db.Where("user_id = ?", userId).Order("updated_at").Find(&orders).Error; err != nil {
		log.Error("getMyOrders:get orders info fail with error: ", err)
		return nil, err
	}
	var orderDisplay []portal.OrderDisplayData
	for _, order := range orders {
		var tmpOrderDisplay = convertToOrderDisplay(order)
		orderDisplay = append(orderDisplay, tmpOrderDisplay)
	}
	return orderDisplay, nil
}

func convertToOrderDisplay(order storage.Order) portal.OrderDisplayData {
	var tmpOrderDisplay portal.OrderDisplayData
	tmpOrderDisplay.OrderId = order.OrderId
	tmpOrderDisplay.OwnerName = order.OwnerName
	tmpOrderDisplay.UserName = order.UserName
	tmpOrderDisplay.Address = order.Address
	tmpOrderDisplay.CreatedAt = order.CreatedAt
	tmpOrderDisplay.Memo = order.Memo
	tmpOrderDisplay.PhoneNumber = order.PhoneNumber

	var productPaymentsDisp storage.ProductPaymentsDisplay
	for _, productPayment := range order.ProductPayments {
		var productPaymentDisp storage.ProductPaymentDisplay
		productPaymentDisp.ProductId = productPayment.ProductId
		productPaymentDisp.ProductName = productPayment.ProductName
		productPaymentDisp.Quantity = productPayment.Quantity
		productPaymentDisp.Price = productPayment.Price
		productPaymentDisp.Currency = productPayment.Currency
		productPaymentDisp.Amount = productPayment.Amount
		productPaymentDisp.AvatarBase64 = utils.ConvertImageToBase64(productPayment.Avatar)
		productPaymentsDisp = append(productPaymentsDisp, productPaymentDisp)
	}
	tmpOrderDisplay.ProductPaymentsDisplay = productPaymentsDisp
	return tmpOrderDisplay
}
