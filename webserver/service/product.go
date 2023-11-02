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

func (s *Service) UpdateSingleProduct(product storage.Product) (storage.Product, error) {
	tx := s.db.Begin()

	if err := tx.Save(&product).Error; err != nil {
		tx.Rollback()
		log.Error("UpdateProduct:save product fail with error: ", err)
		return product, err
	}

	tx.Commit()
	return product, nil
}

func (s *Service) CreateProduct(userId uint64, ownerName string, request portal.CreateProductForm) (*storage.Product, error) {
	product := storage.Product{
		ProductCode: request.ProductCode,
		ProductName: request.ProductName,
		Description: request.Description,
		OwnerId:     userId,
		OwnerName:   ownerName,
		Currency:    request.Currency,
		Price:       request.Price,
		Stock:       request.Stock,
		Avatar:      request.Avatar,
		Images:      request.Images,
		Status:      uint32(utils.Active),
	}

	if err := s.db.Save(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

// Sync Shop related data when user Display name was changed
func (s *Service) SyncShopUserInfo(db *gorm.DB, uID int, displayName, userName string, oldDisplayName string) error {
	// Update Product owner on Product table
	updateProductOwnerBuilder := db.Model(&storage.Product{}).
		Where("owner_id = ?", uID)

	if !utils.IsEmpty(displayName) {
		if err := updateProductOwnerBuilder.UpdateColumn("owner_name", displayName).Error; err != nil {
			return err
		}
	} else if !utils.IsEmpty(userName) && utils.IsEmpty(oldDisplayName) {
		if err := updateProductOwnerBuilder.UpdateColumn("owner_name", userName).Error; err != nil {
			return err
		}
	}
	return nil
}
