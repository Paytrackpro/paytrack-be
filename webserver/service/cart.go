package service

import (
	"fmt"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"gorm.io/gorm"
)

func (s *Service) GetCartsForUser(userId uint64) ([]storage.Cart, error) { // if utils.IsEmpty(f.Sort.Order) {
	var carts []storage.Cart
	if err := s.db.Where("user_id = ?", userId).Order("updated_at").Find(&carts).Error; err != nil {
		log.Error("getCarts:get carts info fail with error: ", err)
		return carts, err
	}

	return carts, nil
}

func (s *Service) CountCartForUser(userId uint64) (int64, error) {
	var count int64
	buildCount := s.db.Model(&storage.Cart{}).Where("user_id = ?", userId)
	if err := buildCount.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) UpdateCart(userId uint64, cartForm portal.CartForm) (storage.Cart, error) {
	var cart storage.Cart
	if err := s.db.Where("user_id = ? AND product_id = ?", userId, cartForm.ProductId).First(&cart).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return cart, utils.NewError(fmt.Errorf("Cart not found"), utils.ErrorNotFound)
		}
		log.Error("UpdateCart:get product fail with error: ", err)
		return cart, err
	}

	utils.SetValue(&cart.Quantity, cartForm.Quantity)
	utils.SetValue(&cart.UpdatedAt, time.Now())

	tx := s.db.Begin()

	if err := tx.Where("user_id = ? AND product_id = ?", userId, cartForm.ProductId).Save(&cart).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateCart:save cart fail with error: ", err)
		return cart, err
	}

	tx.Commit()
	return cart, nil
}

func (s *Service) UpdateSingleCart(cart storage.Cart) (storage.Cart, error) {
	tx := s.db.Begin()

	if err := tx.Save(&cart).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateProduct:save product fail with error: ", err)
		return cart, err
	}

	tx.Commit()
	return cart, nil
}

func (s *Service) AddToCart(userId uint64, request portal.CartForm) (*storage.Cart, error) {

	var existCart storage.Cart
	var isUpdate = false
	if err := s.db.Where("user_id = ? AND product_id = ?", userId, request.ProductId).First(&existCart).Error; err == nil {
		isUpdate = true
	}

	var product storage.Product
	if err := s.db.Where("id = ?", request.ProductId).First(&product).Error; err != nil {
		return &existCart, err
	}

	var quantity = request.Quantity
	var cart storage.Cart
	tx := s.db.Begin()
	if isUpdate {
		quantity += existCart.Quantity
		if quantity > product.Stock {
			quantity = product.Stock
		}
		existCart.Quantity = quantity
		existCart.UpdatedAt = time.Now()
		if err := tx.Where("user_id = ? AND product_id = ?", userId, request.ProductId).Save(&existCart).Error; err != nil {
			tx.Rollback()
			log.Error("AddToCart:save exist cart fail with error: ", err)
			return &existCart, err
		}
		tx.Commit()
		return &existCart, nil
	}

	cart = storage.Cart{
		UserId:    userId,
		OwnerId:   request.OwnerId,
		OwnerName: request.OwnerName,
		ProductId: request.ProductId,
		Quantity:  request.Quantity,
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(&cart).Error; err != nil {
		return nil, err
	}
	tx.Commit()
	return &cart, nil
}
