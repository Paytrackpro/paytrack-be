package service

import (
	"fmt"

	"github.com/Paytrackpro/paytrack-be/storage"
	"github.com/Paytrackpro/paytrack-be/utils"
	"github.com/Paytrackpro/paytrack-be/webserver/portal"
	"gorm.io/gorm"
)

// ValidatePaymentAddress validates the address for a given coin and network
func (s *Service) ValidatePaymentAddress(req portal.ValidateAddressRequest) portal.ValidateAddressResponse {
	// Check if the coin-network combination is supported
	if !utils.IsCoinNetworkSupported(req.Coin, req.Network) {
		return portal.ValidateAddressResponse{
			IsValid: false,
			Reason:  fmt.Sprintf("unsupported coin-network combination: %s on %s", req.Coin, req.Network),
		}
	}

	// Convert network code to Network type
	network := utils.NetworkFromCode(req.Network)

	err := utils.VerifyAddress(req.Address, network)
	if err != nil {
		networkInfo := network.Info()
		return portal.ValidateAddressResponse{
			IsValid: false,
			Reason:  fmt.Sprintf("Invalid address for %s network", networkInfo.Name),
		}
	}

	return portal.ValidateAddressResponse{
		IsValid: true,
	}
}

// CreatePaymentMethod creates a new payment method for the user
func (s *Service) CreatePaymentMethod(userId uint64, req portal.CreatePaymentMethodRequest) (*storage.UserPaymentMethod, error) {
	// Validate the address first
	validateReq := portal.ValidateAddressRequest{
		Coin:    req.Coin,
		Network: req.Network,
		Address: req.Address,
	}
	validation := s.ValidatePaymentAddress(validateReq)
	if !validation.IsValid {
		return nil, fmt.Errorf("address validation failed: %s", validation.Reason)
	}

	// Create the payment method
	paymentMethod := &storage.UserPaymentMethod{
		UserId:  userId,
		Label:   req.Label,
		Coin:    req.Coin,
		Network: req.Network,
		Address: req.Address,
	}

	if err := s.db.Create(paymentMethod).Error; err != nil {
		log.Error("CreatePaymentMethod: failed to create payment method", err)
		return nil, err
	}

	return paymentMethod, nil
}

// GetPaymentMethods returns all payment methods for a user
func (s *Service) GetPaymentMethods(userId uint64) ([]storage.UserPaymentMethod, error) {
	var methods []storage.UserPaymentMethod
	if err := s.db.Where("user_id = ?", userId).Order("created_at DESC").Find(&methods).Error; err != nil {
		log.Error("GetPaymentMethods: failed to get payment methods", err)
		return nil, err
	}
	return methods, nil
}

// GetPaymentMethod returns a specific payment method
func (s *Service) GetPaymentMethod(id, userId uint64) (*storage.UserPaymentMethod, error) {
	var method storage.UserPaymentMethod
	if err := s.db.Where("id = ? AND user_id = ?", id, userId).First(&method).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment method not found")
		}
		log.Error("GetPaymentMethod: failed to get payment method", err)
		return nil, err
	}
	return &method, nil
}

// UpdatePaymentMethod updates a payment method (only label can be updated)
func (s *Service) UpdatePaymentMethod(id, userId uint64, req portal.UpdatePaymentMethodRequest) (*storage.UserPaymentMethod, error) {
	method, err := s.GetPaymentMethod(id, userId)
	if err != nil {
		return nil, err
	}

	method.Label = req.Label
	if err := s.db.Save(method).Error; err != nil {
		log.Error("UpdatePaymentMethod: failed to update payment method", err)
		return nil, err
	}

	return method, nil
}

// DeletePaymentMethod deletes a payment method
func (s *Service) DeletePaymentMethod(id, userId uint64) error {
	// Check if the payment method exists and belongs to the user
	method, err := s.GetPaymentMethod(id, userId)
	if err != nil {
		return err
	}

	// Check if this payment method is used in any payments
	var count int64
	if err := s.db.Model(&storage.Payment{}).Where("user_payment_method_id = ?", id).Count(&count).Error; err != nil {
		log.Error("DeletePaymentMethod: failed to check payment usage", err)
		return err
	}

	if count > 0 {
		return fmt.Errorf("cannot delete payment method: it is used in %d payment(s)", count)
	}

	// Delete the payment method
	if err := s.db.Delete(method).Error; err != nil {
		log.Error("DeletePaymentMethod: failed to delete payment method", err)
		return err
	}

	return nil
}

// GetSupportedNetworks returns all supported coin and network combinations
func (s *Service) GetSupportedNetworks() portal.SupportedNetworksResponse {
	return portal.SupportedNetworksResponse{
		SupportedCoins: utils.GetSupportedCoinsAndNetworks(),
	}
}

// GetSettings returns comprehensive settings information for clients
func (s *Service) GetSettings() portal.SettingsResponse {
	return portal.SettingsResponse{
		SupportedCoins: utils.GetSupportedCoinsAndNetworks(),
		// Add other settings here as needed in the future
	}
}
