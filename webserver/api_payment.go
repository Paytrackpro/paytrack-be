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
		a.errorResponse(w, err, http.StatusBadRequest)
		return
	}
	claims, _ := a.credentialsInfo(r)
	payment := f.Payment(claims.Id)
	if err = a.db.Create(&payment); err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	var response = Map{
		"payment": payment,
	}
	accessToken, err := a.crypto.Encrypt(utils.PaymentPlainText(payment.Id))
	response["generateToken"] = err == nil
	if err != nil {
		response["generateTokenError"] = err.Error()
		a.successResponse(w, response)
		return
	}
	response["token"] = accessToken
	if payment.ContactMethod == storage.PaymentTypeEmail {
		err := a.mail.Send("Payment request", "paymentNotify", email.PaymentNotifyVar{
			Title:     "Payment request",
			Sender:    payment.SenderEmail,
			Requester: claims.UserName,
			Link:      a.conf.ClientAddr,
		})
		response["mailNotification"] = err == nil
		if err != nil {
			response["mailNotificationError"] = err.Error()
		}
	}
	// todo: do we have to notify with internal case?
	// setup notification system
	a.successResponse(w, response)
}

func (a *apiPayment) getPayment(w http.ResponseWriter, r *http.Request) {
	var id = chi.URLParam(r, "id")
	r.ParseForm()
	var token = r.Form.Get("token")
	var payment storage.Payment
	if err := a.db.GetById(id, &payment); err != nil {
		a.errorResponse(w, err, http.StatusNotFound)
		return
	}
	if err := a.verifyAccessPayment(token, payment, r); err != nil {
		a.errorResponse(w, err, http.StatusForbidden)
		return
	}
	a.successResponse(w, payment)
}

func (a *apiPayment) verifyAccessPayment(token string, payment storage.Payment, r *http.Request) error {
	claims, _ := a.credentialsInfo(r)
	if payment.ContactMethod == storage.PaymentTypeInternal && claims.Id != payment.SenderId {
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
		a.errorResponse(w, err, http.StatusBadRequest)
		return
	}
	var payment storage.Payment
	if err := a.db.GetById(f.Id, &payment); err != nil {
		a.errorResponse(w, err, http.StatusNotFound)
		return
	}
	if err := a.verifyAccessPayment(f.Token, payment, r); err != nil {
		a.errorResponse(w, err, http.StatusForbidden)
		return
	}
	f.Process(&payment)
	if err = a.db.Save(&payment); err != nil {
		a.errorResponse(w, err, http.StatusInternalServerError)
		return
	}
	a.successResponse(w, payment)
}
