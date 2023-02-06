package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/email"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type apiPayment struct {
	*WebServer
}

func (a *apiPayment) createPayment(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentRequest
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	claims, _ := a.credentialsInfo(r)
	payment := f.Payment(claims.Id)
	if err = a.db.Create(&payment); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	var response = Map{
		"payment": payment,
	}
	accessToken, _ := a.crypto.Encrypt(utils.PaymentPlainText(payment.Id))
	response["token"] = accessToken
	var customErr *utils.Error
	if payment.ContactMethod == storage.PaymentTypeEmail {
		err := a.mail.Send("Payment request", "paymentNotify", email.PaymentNotifyVar{
			Title:     "Payment request",
			Sender:    payment.SenderEmail,
			Requester: claims.UserName,
			Link:      a.conf.ClientAddr,
		}, payment.SenderEmail)
		if err != nil {
			customErr = utils.SendMailFailed.With(err)
		}
	}
	// todo: do we have to notify with internal case?
	// setup notification system
	utils.ResponseOK(w, response, customErr)
}

func (a *apiPayment) getPayment(w http.ResponseWriter, r *http.Request) {
	var id = chi.URLParam(r, "id")
	var token = r.FormValue("token")
	var payment storage.Payment
	if err := a.db.GetById(id, &payment); err != nil {
		utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
		return
	}
	if err := a.verifyAccessPayment(token, payment, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}
	utils.ResponseOK(w, payment)
}

func (a *apiPayment) verifyAccessPayment(token string, payment storage.Payment, r *http.Request) error {
	claims, _ := a.parseBearer(r)
	if claims != nil && claims.Id == payment.RequesterId {
		return nil
	}
	if payment.ContactMethod == storage.PaymentTypeInternal && (claims == nil || claims.Id != payment.SenderId) {
		return fmt.Errorf("only requested user has the access to process the payment")
	}
	if payment.ContactMethod == storage.PaymentTypeEmail {
		var plainText, err = a.crypto.Decrypt(token)
		if err != nil {
			return err
		}
		if plainText != utils.PaymentPlainText(payment.Id) {
			return fmt.Errorf("the token is invalid")
		}
	}
	return nil
}

func (a *apiPayment) processPayment(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentConfirm
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	var payment storage.Payment
	if err := a.db.GetById(f.Id, &payment); err != nil {
		utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
		return
	}
	if err := a.verifyAccessPayment(f.Token, payment, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}
	if payment.Status == storage.PaymentStatusPaid {
		utils.Response(w, http.StatusBadRequest,
			utils.NewError(fmt.Errorf("payment was processed"), utils.ErrorBadRequest), nil)
		return
	}
	f.Process(&payment)
	if err = a.db.Save(&payment); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	utils.ResponseOK(w, payment)
}

func (a *apiPayment) listPayments(w http.ResponseWriter, r *http.Request) {
	var f storage.PaymentFilter
	if err := a.parseQueryAndValidate(r, &f); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	// checking error on claims is not needed since listPayments is for logged in api,
	// the checking is from the logged in middleware
	claims, _ := a.parseBearer(r)
	if claims.UserRole != utils.UserRoleAdmin {
		f.SenderIds = append(f.SenderIds, claims.Id)
		f.RequesterIds = append(f.RequesterIds, claims.Id)
	}
	var payments []storage.Payment
	if err := a.db.GetList(&f, &payments); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	utils.ResponseOK(w, payments)
}
