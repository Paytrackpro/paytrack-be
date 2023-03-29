package webserver

import (
	"fmt"
	"net/http"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/email"
	paymentService "code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-chi/chi/v5"
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
	if err := a.verifyAccessPayment(f.Token, payment, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}
	var oldStatus = payment.Status
	var userId uint64
	claims, _ := a.parseBearer(r)
	if claims != nil {
		userId = claims.Id
	}
	err = f.Payment(userId, &payment, false)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	if err = a.db.Save(&payment); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	if payment.ReceiverId != claims.Id && payment.SenderId != claims.Id {
		// Not approval
		if len(payment.Approvers) == 0 {
			payment.Status = storage.PaymentStatusWaitApproval
		} else {
			payment.Status = storage.PaymentStatusWaitApproval

			// find record approval of user
			for _, ap := range payment.Approvers {
				if ap.ApproverId == claims.Id {
					payment.Status = storage.PaymentStatusApproved
				}
			}
		}
	} else {
		if payment.SenderId == claims.Id {
			// for sender
			if payment.Status == storage.PaymentStatusConfirmed || payment.Status == storage.PaymentStatusApproved {
				payment.Status = storage.PaymentStatusSent
			}
		} else {

			approvers, err := a.service.GetApproverForPayment(payment.SenderId, payment.ReceiverId)
			if err != nil {
				utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
				return
			}
			payment.IsApproved = len(approvers) <= len(payment.Approvers)

			// for receiver
			if payment.Status != storage.PaymentStatusConfirmed && payment.Status != storage.PaymentStatusRejected && payment.Approvers != nil && payment.Status != storage.PaymentStatusApproved && payment.Status != storage.PaymentStatusPaid {
				payment.Status = storage.PaymentStatusWaitApproval
			}
		}
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
	if claims == nil {
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
			Path:      fmt.Sprintf("/payment/%d/%s", p.Id, accessToken),
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

	approvers, err := a.service.GetApproverForPayment(f.SenderId, f.ReceiverId)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}

	var payment storage.Payment
	err = f.Payment(claims.Id, &payment, len(approvers) > 0)
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
	if err := a.verifyAccessPayment(token, payment, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}

	if token == "" {
		claims, _ := a.parseBearer(r)
		if payment.ReceiverId != claims.Id && payment.SenderId != claims.Id {
			// Not approval
			if len(payment.Approvers) == 0 {
				payment.Status = storage.PaymentStatusWaitApproval
			} else {
				payment.Status = storage.PaymentStatusWaitApproval

				// find record approval of user
				for _, ap := range payment.Approvers {
					if ap.ApproverId == claims.Id {
						payment.Status = storage.PaymentStatusApproved
					}
				}
			}
		} else {
			if payment.SenderId == claims.Id {
				// for sender
				if payment.Status == storage.PaymentStatusConfirmed || payment.Status == storage.PaymentStatusApproved {
					payment.Status = storage.PaymentStatusSent
				}
			} else {

				approvers, err := a.service.GetApproverForPayment(payment.SenderId, payment.ReceiverId)
				if err != nil {
					utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
					return
				}
				payment.IsApproved = len(approvers) <= len(payment.Approvers)

				// for receiver
				if payment.Status != storage.PaymentStatusConfirmed && payment.Status != storage.PaymentStatusRejected && payment.Approvers != nil && payment.Status != storage.PaymentStatusApproved && payment.Status != storage.PaymentStatusPaid {
					payment.Status = storage.PaymentStatusWaitApproval
				}
			}
		}
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

	approver, err := a.service.GetApprovalSetting(payment.SenderId, payment.ReceiverId, claims.Id)
	if err != nil {
		return err
	}

	if claims.Id == payment.SenderId || (claims.Id == payment.ReceiverId && payment.Status != storage.PaymentStatusCreated) || approver != nil {
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
	claims, _ := a.parseBearer(r)
	if p.ReceiverId != claims.Id && p.SenderId != claims.Id {
		// Not approval
		if len(p.Approvers) == 0 {
			p.Status = storage.PaymentStatusWaitApproval
		} else {
			p.Status = storage.PaymentStatusWaitApproval

			// find record approval of user
			for _, ap := range p.Approvers {
				if ap.ApproverId == claims.Id {
					p.Status = storage.PaymentStatusApproved
				}
			}
		}
	} else {
		if p.SenderId == claims.Id {
			// for sender
			if p.Status == storage.PaymentStatusConfirmed || p.Status == storage.PaymentStatusApproved {
				p.Status = storage.PaymentStatusSent
			}
		} else {
			// for receiver
			if p.Status != storage.PaymentStatusConfirmed && p.Status != storage.PaymentStatusRejected && p.Approvers != nil && p.Status != storage.PaymentStatusApproved && p.Status != storage.PaymentStatusPaid {
				p.Status = storage.PaymentStatusWaitApproval
			}
		}
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
	if payment.ContactMethod == storage.PaymentTypeInternal {
		if claims, _ := a.parseBearer(r); !(claims != nil && claims.Id == payment.ReceiverId) {
			utils.Response(w, http.StatusForbidden,
				utils.NewError(fmt.Errorf("you do not have access right"), utils.ErrorForbidden), nil)
			return
		}
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
	switch f.RequestType {
	case storage.PaymentTypeReminder:
		f.ReceiverIds = []uint64{claims.Id}
		f.Statuses = []storage.PaymentStatus{
			storage.PaymentStatusSent,
			storage.PaymentStatusConfirmed,
			storage.PaymentStatusPaid,
			storage.PaymentStatusApproved,
			storage.PaymentStatusRejected,
		}
	case storage.PaymentTypeRequest:
		f.SenderIds = []uint64{claims.Id}
	default:
		if claims.UserRole != utils.UserRoleAdmin {
			f.SenderIds = append(f.SenderIds, claims.Id)
			f.ReceiverIds = append(f.ReceiverIds, claims.Id)
		}
	}

	if f.RequestType == storage.PaymentTypeReminder {
		approvers, err := a.service.GetSettingOfApprover(claims.Id)
		if err != nil {
			utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
			return
		}
		f.Approvers = approvers
	}

	var payments []storage.Payment
	if err := a.db.GetList(&f, &payments); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}

	// use for receiver and approver
	if f.RequestType == storage.PaymentTypeReminder {
		for i, pay := range payments {
			// for approver
			if pay.ReceiverId != claims.Id {
				// Not approval
				if len(pay.Approvers) == 0 {
					payments[i].Status = storage.PaymentStatusWaitApproval
				} else {
					payments[i].Status = storage.PaymentStatusWaitApproval
					// find record approval of user
					for _, ap := range pay.Approvers {
						if ap.ApproverId == claims.Id {
							payments[i].Status = storage.PaymentStatusApproved
						}
					}
				}
			} else {
				// for receiver
				if pay.Status != storage.PaymentStatusConfirmed && pay.Status != storage.PaymentStatusRejected && pay.Approvers != nil && pay.Status != storage.PaymentStatusApproved && pay.Status != storage.PaymentStatusPaid {
					payments[i].Status = storage.PaymentStatusWaitApproval
				}
			}
		}
	} else {
		// use for sender
		for i, pay := range payments {
			if pay.Status == storage.PaymentStatusConfirmed || pay.Status == storage.PaymentStatusApproved {
				payments[i].Status = storage.PaymentStatusSent
			}
		}
	}

	count, _ := a.db.Count(&f, &storage.Payment{})
	utils.ResponseOK(w, Map{
		"payments": payments,
		"count":    count,
	})
}

func (a *apiPayment) approveRequest(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.parseBearer(r)
	var f portal.ApprovalRequest
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	payment, err := a.service.ApprovePaymentRequest(f.PaymentId, claims.Id, claims.UserName)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	utils.ResponseOK(w, payment)
}
func (a *apiPayment) rejectPayment(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentReject
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

	if err := a.verifyAccessPayment(f.Token, payment, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}

	if payment.ContactMethod == storage.PaymentTypeInternal {
		if claims, _ := a.parseBearer(r); !(claims != nil && claims.Id == payment.ReceiverId) {
			utils.Response(w, http.StatusForbidden,
				utils.NewError(fmt.Errorf("you do not have access right"), utils.ErrorForbidden), nil)
			return
		}
	}

	if payment.Status == storage.PaymentStatusPaid {
		utils.Response(w, http.StatusBadRequest,
			utils.NewError(fmt.Errorf("payment was processed"), utils.ErrorBadRequest), nil)
		return
	}

	payment.Status = storage.PaymentStatusRejected
	payment.RejectionReason = f.RejectionReason
	if err = a.db.Save(&payment); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}

	utils.ResponseOK(w, payment)
}
