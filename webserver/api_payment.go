package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/email"
	paymentService "code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

type apiPayment struct {
	*WebServer
}

// updatePayment user can update the payment when the status still be created
func (a *apiPayment) updatePayment(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentRequest
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	var id = chi.URLParam(r, "id")
	var payment storage.Payment
	var filter = storage.PaymentFilter{
		Ids: []uint64{utils.Uint64(id)},
	}
	if err := a.db.First(&filter, &payment); err != nil {
		utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
		return
	}
	if payment.Status == storage.PaymentStatusPaid {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("the payment was marked as paid"), utils.ErrorBadRequest), nil)
		return
	}
	var oldStatus = payment.Status
	claims, _ := a.credentialsInfo(r)
	err = f.Payment(claims.Id, &payment)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	if err = a.db.Save(&payment); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	accessToken, customErr := a.sendNotification(oldStatus, payment, claims)
	utils.ResponseOK(w, Map{
		"payment": payment,
		"token":   accessToken,
	}, customErr)
}

func (a *apiPayment) sendNotification(oldStatus storage.PaymentStatus, p storage.Payment, claims *authClaims) (string, *utils.Error) {
	if !(oldStatus == storage.PaymentStatusCreated && p.Status == storage.PaymentStatusSent) {
		return "", nil
	}
	accessToken, _ := a.crypto.Encrypt(utils.PaymentPlainText(p.Id))
	var customErr *utils.Error
	if p.ContactMethod == storage.PaymentTypeEmail {
		err := a.mail.Send("Payment request", "paymentNotify", email.PaymentNotifyVar{
			Title:     "Payment request",
			Receiver:  p.ExternalEmail,
			Sender:    claims.UserName,
			Link:      a.conf.ClientAddr,
			Path:      fmt.Sprintf("/payment/%d?token=%s", p.Id, accessToken),
			IsRequest: claims.Id == p.ReceiverId,
		}, p.ExternalEmail)
		if err != nil {
			customErr = utils.SendMailFailed.With(err)
		}
	}
	// todo: do we have to notify with internal case?
	// setup notification system
	return accessToken, customErr
}

func (a *apiPayment) createPayment(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentRequest
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	claims, _ := a.credentialsInfo(r)
	var payment storage.Payment
	err = f.Payment(claims.Id, &payment)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	if err = a.db.Create(&payment); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	accessToken, customErr := a.sendNotification(storage.PaymentStatusCreated, payment, claims)
	utils.ResponseOK(w, Map{
		"payment": payment,
		"token":   accessToken,
	}, customErr)
}

func (a *apiPayment) getPayment(w http.ResponseWriter, r *http.Request) {
	var id = chi.URLParam(r, "id")
	var token = r.FormValue("token")
	var payment storage.Payment
	var f = storage.PaymentFilter{
		Ids: []uint64{utils.Uint64(id)},
	}
	if err := a.db.First(&f, &payment); err != nil {
		utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
		return
	}
	// if the user is the creator
	claims, _ := a.parseBearer(r)
	if claims != nil && claims.Id == payment.ReceiverId {
		utils.ResponseOK(w, payment)
		return
	}
	// checking if the user is the receiver
	if err := a.verifyAccessPayment(token, payment, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}
	utils.ResponseOK(w, payment)
}

// verifyAccessPayment checking if the user is the requested user
func (a *apiPayment) verifyAccessPayment(token string, payment storage.Payment, r *http.Request) error {
	claims, _ := a.parseBearer(r)
	if claims == nil {
		if payment.ContactMethod == storage.PaymentTypeEmail {
			var plainText, err = a.crypto.Decrypt(token)
			if err != nil {
				return err
			}
			if plainText != utils.PaymentPlainText(payment.Id) {
				return fmt.Errorf("the token is invalid")
			}
			return nil
		}
		return fmt.Errorf("you do not have access")
	}
	if claims.Id == payment.SenderId || claims.Id == payment.ReceiverId {
		return nil
	}
	return fmt.Errorf("you do not have access")
}

// requestRate used for the requested user to request the cryptocurrency rate with USDT
func (a *apiPayment) requestRate(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentRequestRate
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	var p storage.Payment
	var filter = storage.PaymentFilter{
		Ids: []uint64{f.Id},
	}
	if err := a.db.First(&filter, &p); err != nil {
		utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
		return
	}
	if p.Status == storage.PaymentStatusPaid {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("the payment was marked as paid"), utils.ErrorBadRequest), nil)
		return
	}
	// only the requested user has the access to process the payment
	if err := a.verifyAccessPayment(f.Token, p, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}
	price, err := paymentService.GetPrice(f.PaymentMethod)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	p.PaymentMethod = f.PaymentMethod
	p.PaymentAddress = f.PaymentAddress
	p.ConvertRate = price
	p.ConvertTime = time.Now()
	p.ExpectedAmount = utils.BtcRoundFloat(p.Amount / price)
	if err = a.db.Save(&p); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	utils.ResponseOK(w, p)
}

func (a *apiPayment) processPayment(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentConfirm
	err := a.parseJSONAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	var payment storage.Payment
	var filter = storage.PaymentFilter{
		Ids: []uint64{f.Id},
	}
	if err := a.db.First(&filter, &payment); err != nil {
		utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
		return
	}
	if payment.Status == storage.PaymentStatusPaid {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("the payment was marked as paid"), utils.ErrorBadRequest), nil)
		return
	}
	// only the requested user has the access to process the payment
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
	count, _ := a.db.Count(&f, &storage.Payment{})
	utils.ResponseOK(w, Map{
		"payments": payments,
		"count":    count,
	})
}
