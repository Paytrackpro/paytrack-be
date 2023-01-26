package webserver

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
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
	// todo: send email or notify the sender
	a.successResponse(w, payment)
}

func (a *apiPayment) getPayment(w http.ResponseWriter, r *http.Request) {
	var id = chi.URLParam(r, "id")
	var payment storage.Payment
	if err := a.db.GetById(id, &payment); err != nil {
		a.errorResponse(w, err, http.StatusNotFound)
		return
	}
	if err := a.verifyAccessPayment(payment, r); err != nil {
		a.errorResponse(w, err, http.StatusForbidden)
		return
	}
	a.successResponse(w, payment)
}

func (a *apiPayment) verifyAccessPayment(payment storage.Payment, r *http.Request) error {
	claims, _ := a.credentialsInfo(r)
	if payment.ContactMethod == storage.PaymentTypeInternal && claims.Id != payment.SenderId {
		return fmt.Errorf("only requested user has the access to process the payment")
	}
	if payment.ContactMethod == storage.PaymentTypeEmail {
		// todo: verify here when send email is supported
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
	if err := a.verifyAccessPayment(payment, r); err != nil {
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
