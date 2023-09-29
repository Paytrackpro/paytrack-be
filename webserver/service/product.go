package service

import (
	"fmt"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"gorm.io/gorm"
)

func (s *Service) GetProductInfo(id uint64) (storage.Product, error) {
	var product storage.Product
	if err := s.db.Where("id = ?", id).Find(&product).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return product, utils.NewError(fmt.Errorf("product not found"), utils.ErrorNotFound)
		}
		log.Error("getProductInfo:get product info fail with error: ", err)
		return product, err
	}
	return product, nil
}

func (s *Service) UpdateProduct(id uint64, productInfo portal.UpdateProductRequest) (storage.Product, error) {
	var product storage.Product
	if err := s.db.Where("id = ?", id).First(&product).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return product, utils.NewError(fmt.Errorf("Product not found"), utils.ErrorNotFound)
		}
		log.Error("UpdateProductInfo:get product fail with error: ", err)
		return product, err
	}

	utils.SetValue(&product.ProductCode, productInfo.ProductCode)
	utils.SetValue(&product.ProductName, productInfo.ProductName)
	utils.SetValue(&product.Description, productInfo.Description)
	utils.SetValue(&product.Currency, productInfo.Currency)
	utils.SetValue(&product.Avatar, productInfo.Avatar)
	utils.SetValue(&product.Images, productInfo.Images)
	utils.SetValue(&product.Price, productInfo.Price)
	utils.SetValue(&product.Stock, productInfo.Stock)
	utils.SetValue(&product.Status, productInfo.Status)

	tx := s.db.Begin()

	if err := tx.Save(&product).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateProduct:save product fail with error: ", err)
		return product, err
	}

	tx.Commit()
	return product, nil
}

func (s *Service) CreateProduct(userId uint64, request portal.CreateProductForm) (*storage.Product, error) {
	product := storage.Product{
		ProductCode: request.ProductCode,
		ProductName: request.ProductName,
		Description: request.Description,
		OwnerId:     userId,
		Currency:    request.Currency,
		Price:       request.Price,
		Stock:       request.Stock,
		Avatar:      request.Avatar,
		Images:      request.Images,
		Status:      1,
	}

	if err := s.db.Save(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}
