package webserver

import (
	"fmt"
	"net/http"
	"strconv"

	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-chi/chi/v5"
)

type apiPaymentMethod struct {
	*WebServer
}

// createPaymentMethod handles POST /api/user/payment-methods
func (a *apiPaymentMethod) createPaymentMethod(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	var req portal.CreatePaymentMethodRequest
	if err := a.parseJSONAndValidate(r, &req); err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	paymentMethod, err := a.service.CreatePaymentMethod(claims.Id, req)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	utils.Response(w, http.StatusCreated, nil, paymentMethod)
}

// getPaymentMethods handles GET /api/user/payment-methods
func (a *apiPaymentMethod) getPaymentMethods(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	methods, err := a.service.GetPaymentMethods(claims.Id)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, methods)
}

// updatePaymentMethod handles PUT /api/user/payment-methods/{id}
func (a *apiPaymentMethod) updatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("invalid payment method id"), nil)
		return
	}

	var req portal.UpdatePaymentMethodRequest
	if err := a.parseJSONAndValidate(r, &req); err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	paymentMethod, err := a.service.UpdatePaymentMethod(id, claims.Id, req)
	if err != nil {
		if err.Error() == "payment method not found" {
			utils.Response(w, http.StatusNotFound, err, nil)
		} else {
			utils.Response(w, http.StatusInternalServerError, err, nil)
		}
		return
	}

	utils.ResponseOK(w, paymentMethod)
}

// deletePaymentMethod handles DELETE /api/user/payment-methods/{id}
func (a *apiPaymentMethod) deletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("invalid payment method id"), nil)
		return
	}

	err = a.service.DeletePaymentMethod(id, claims.Id)
	if err != nil {
		if err.Error() == "payment method not found" {
			utils.Response(w, http.StatusNotFound, err, nil)
		} else if err.Error()[0:12] == "cannot delete" {
			utils.Response(w, http.StatusBadRequest, err, nil)
		} else {
			utils.Response(w, http.StatusInternalServerError, err, nil)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateAddress handles POST /api/user/payment-methods/validate-address
func (a *apiPaymentMethod) validateAddress(w http.ResponseWriter, r *http.Request) {
	var req portal.ValidateAddressRequest
	if err := a.parseJSONAndValidate(r, &req); err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	response := a.service.ValidatePaymentAddress(req)
	utils.ResponseOK(w, response)
}

// getSupportedNetworks handles GET /api/payment-methods/supported-networks
func (a *apiPaymentMethod) getSupportedNetworks(w http.ResponseWriter, r *http.Request) {
	response := a.service.GetSupportedNetworks()
	utils.ResponseOK(w, response)
}