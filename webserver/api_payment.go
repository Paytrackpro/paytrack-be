package webserver

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/email"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"code.cryptopower.dev/mgmt-ng/be/webserver/service"
	"github.com/go-chi/chi/v5"
)

type apiPayment struct {
	*WebServer
}

// Use for sender and receiver
func (a *apiPayment) updatePayment(w http.ResponseWriter, r *http.Request) {
	var body portal.PaymentRequest
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	var strId = chi.URLParam(r, "id")
	paymentId := utils.Uint64(strId)
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}

	if claims == nil {
		// receiver is external
		if utils.IsEmpty(body.Token) {
			utils.Response(w, http.StatusForbidden, utils.NewError(fmt.Errorf("do not have access"), utils.ErrorBadRequest), nil)
			return
		}

		if err := a.verifyTokenPayment(body.Token, paymentId); err != nil {
			log.Error(err)
			utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
			return
		}
		payment, err := a.service.UpdatePayment(paymentId, 0, body)
		if err != nil {
			log.Error(err)
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}

		utils.ResponseOK(w, Map{
			"payment": payment,
			"token":   "",
		}, nil)
	} else if body.SenderId == claims.Id || body.ReceiverId == claims.Id {
		// sender and receiver update
		payment, err := a.service.UpdatePayment(paymentId, claims.Id, body)
		if err != nil {
			log.Error(err)
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}
		if body.Status == storage.PaymentStatusSent || body.Status == storage.PaymentStatusCreated {
			a.reloadList([]string{fmt.Sprint(body.ReceiverId)}, "")
		}
		utils.ResponseOK(w, Map{
			"payment": payment,
			"token":   "",
		}, nil)
	} else {
		utils.Response(w, http.StatusForbidden, utils.NewError(fmt.Errorf("do not have access"), utils.ErrorBadRequest), nil)
	}
}

func (a *apiPayment) reloadList(rooms []string, data interface{}) {
	for _, room := range rooms {
		a.socket.BroadcastToRoom("", room, "reloadList", data)
	}
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
			log.Error(err)
			customErr = utils.SendMailFailed.With(err)
		}
	}
	// todo: do we have to notify with internal case?
	// setup notification system
	return accessToken, customErr
}

func (a *apiPayment) createPayment(w http.ResponseWriter, r *http.Request) {
	var body portal.PaymentRequest
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	userInfo, _ := a.credentialsInfo(r)

	//validate body
	if body.SenderId != userInfo.Id {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("the sender must be you"), nil)
		return
	}

	if body.ReceiverId == userInfo.Id {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("the receiver must be someone else"), nil)
		return
	}

	// sender only save payment as draft or sent to receiver
	if body.Status != storage.PaymentStatusCreated && body.Status != storage.PaymentStatusSent {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("you only save it as draft or sent it to receiver"), nil)
		return
	}

	payment, err := a.service.CreatePayment(userInfo.Id, userInfo.UserName, userInfo.DisplayName,
		userInfo.ShowDraftForRecipient, body)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusOK, err, nil)
		return
	}
	res := Map{
		"payment": payment,
	}
	a.reloadList([]string{fmt.Sprint(payment.ReceiverId)}, "")
	if body.ContactMethod == storage.PaymentTypeEmail {
		token, customErr := a.sendNotification(storage.PaymentStatusCreated, *payment, userInfo)
		res["token"] = token
		utils.ResponseOK(w, res, customErr)
	} else {
		utils.ResponseOK(w, res, nil)
	}

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
	a.sortPaymentDetails(payment)
	if err := a.verifyAccessPayment(token, payment, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}
	utils.ResponseOK(w, payment)
}

func (a *apiPayment) sortPaymentDetails(payment storage.Payment) {
	if len(payment.Details) < 2 {
		return
	}
	for i := 0; i < len(payment.Details); i++ {
		for j := i + 1; j < len(payment.Details); j++ {
			date1, err := time.Parse("2006/01/02", utils.HandlerDateFormat(payment.Details[i].Date))
			if err != nil {
				log.Info(err)
				continue
			}
			date2, err := time.Parse("2006/01/02", utils.HandlerDateFormat(payment.Details[j].Date))
			if err != nil {
				log.Info(err)
				continue
			}
			beforeUnix := date1.Unix()
			afterUnix := date2.Unix()
			if afterUnix < beforeUnix {
				var tmpDetail = payment.Details[i]
				payment.Details[i] = payment.Details[j]
				payment.Details[j] = tmpDetail
			}
		}
	}
}

func (a *apiPayment) getMonthlySummary(w http.ResponseWriter, r *http.Request) {
	var query portal.SummaryFilter
	if err := a.parseQueryAndValidate(r, &query); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusForbidden, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorForbidden), nil)
		return
	}
	paymentSummary, err := a.service.GetRequestSummary(claims.Id, query)
	if err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}
	utils.ResponseOK(w, Map{
		"summary": paymentSummary,
	})
}

func (a *apiPayment) verifyTokenPayment(token string, paymentId uint64) error {
	var plainText, err = a.crypto.Decrypt(token)
	if err != nil {
		return err
	}
	if plainText != utils.PaymentPlainText(paymentId) {
		return fmt.Errorf("the token is invalid")
	}
	return nil
}

// verifyAccessPayment checking if the user is the requested user
func (a *apiPayment) verifyAccessPayment(token string, payment storage.Payment, r *http.Request) error {
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		return fmt.Errorf("Get credentials info failed")
	}
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
	var validApprover = false
	projectIds := make([]string, 0)
	if len(payment.Details) > 0 {
		for _, detail := range payment.Details {
			if detail.ProjectId > 0 {
				projectIdStr := fmt.Sprintf("%d", detail.ProjectId)
				if !slices.Contains(projectIds, projectIdStr) {
					projectIds = append(projectIds, projectIdStr)
				}
			}
		}
	}
	if len(projectIds) > 0 {
		approvers, err := a.service.GetProjectApprovers(projectIds)
		if err != nil {
			return err
		}
		if len(approvers) > 0 {
			isClaimExist := false
			for _, apv := range approvers {
				if apv.MemberId == claims.Id {
					isClaimExist = true
					break
				}
			}
			validApprover = isClaimExist
		}
	}
	if claims.Id == payment.SenderId || (claims.Id == payment.ReceiverId && (payment.Status != storage.PaymentStatusCreated || (payment.Status == storage.PaymentStatusCreated && payment.ShowDraftRecipient))) || validApprover {
		return nil
	}
	return fmt.Errorf("you do not have access")
}

func (a *apiPayment) getBtcBulkRate(w http.ResponseWriter, r *http.Request) {
	rate, err := a.service.GetBTCBulkRate()
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}

	res := portal.GetRateResponse{
		Rate:        rate,
		ConvertTime: time.Now().Unix(),
	}

	utils.ResponseOK(w, res)
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
		utils.ResponseOK(w, Map{
			"rate":           float64(0),
			"convertTime":    time.Now(),
			"expectedAmount": float64(0),
			"isPaid":         true,
		})
		return
	}
	// only the requested user has the access to process the payment
	if err := a.verifyAccessPayment(f.Token, p, r); err != nil {
		utils.Response(w, http.StatusForbidden, utils.NewError(err, utils.ErrorForbidden), nil)
		return
	}
	if utils.IsEmpty(f.Exchange) {
		f.Exchange = service.Binance
	}
	handlerExchange := strings.ToLower(f.Exchange)
	rate, err := a.service.GetExchangeRate(handlerExchange, f.PaymentMethod)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	utils.ResponseOK(w, Map{
		"rate":           rate,
		"convertTime":    time.Now(),
		"expectedAmount": utils.BtcRoundFloat(p.Amount / rate),
	})
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
		claims, isOk := a.credentialsInfo(r)
		if !isOk {
			utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
			return
		}
		if !(claims != nil && claims.Id == payment.ReceiverId) {
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
	a.reloadList([]string{fmt.Sprint(payment.ReceiverId)}, "")
	utils.ResponseOK(w, payment)
}

func (a *apiPayment) deleteDraft(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	payment := &storage.Payment{}
	if err := a.db.GetDB().Where("id = ?", id).First(payment).Error; err == nil {
		a.reloadList([]string{fmt.Sprint(payment.SenderId)}, "")
		a.db.GetDB().Where("id = ?", id).Delete(&storage.Payment{})
	}
	utils.ResponseOK(w, nil)
}

func (a *apiPayment) getInitializationCount(w http.ResponseWriter, r *http.Request) {
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	approvalCount, err1 := a.service.GetApprovalsCount(claims.Id, claims.ShowApproved)
	if err1 != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err1, utils.ErrorInternalCode), nil)
		return
	}

	unpaidCount, err2 := a.service.GetUnpaidCount(claims.Id)
	if err2 != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err2, utils.ErrorInternalCode), nil)
		return
	}

	utils.ResponseOK(w, Map{
		"approvalCount": approvalCount,
		"unpaidCount":   unpaidCount,
	})
}

func (a *apiPayment) listPayments(w http.ResponseWriter, r *http.Request) {
	var query storage.PaymentFilter
	if err := a.parseQueryAndValidate(r, &query); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	//default sortable is createdAt desc (newest before)
	if utils.IsEmpty(query.Sort.Order) {
		query.Sort.Order = "created_at desc"
	}
	if strings.Contains(query.Sort.Order, "updatedAt") {
		query.Sort.Order = strings.ReplaceAll(query.Sort.Order, "updatedAt", "updated_at")
	}
	query.Sort.Order = strings.ReplaceAll(query.Sort.Order, "sentAt", "sent_at")
	query.Sort.Order = strings.ReplaceAll(query.Sort.Order, "receiverName", "receiver_name")
	query.Sort.Order = strings.ReplaceAll(query.Sort.Order, "senderName", "sender_name")
	query.Sort.Order = strings.ReplaceAll(query.Sort.Order, "startDate", "start_date")
	query.Sort.Order = strings.ReplaceAll(query.Sort.Order, "projectName", "project_name")

	if query.RequestType == storage.PaymentTypeBulkPayBTC {
		payments, count, err := a.service.GetBulkPaymentBTC(claims.Id, query.Page, query.Size, query.Sort.Order)
		if err != nil {
			utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
			return
		}

		utils.ResponseOK(w, Map{
			"payments": payments,
			"count":    count,
		})
		return
	}

	payments, count, totalAmountUnpaid, err := a.service.GetListPayments(claims.Id, claims.UserRole, query)
	//payments, count, err := a.service.GetListPayments(claims.Id, claims.UserRole, query)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}

	totalUnpaid := int64(0)
	for _, payment := range payments {
		if payment.Status == 0 {
			totalUnpaid += int64(payment.Amount * 100)
		}
	}
	// finalTotalUnpaid := float64(totalUnpaid) / 100
	utils.ResponseOK(w, Map{
		"payments":    payments,
		"count":       count,
		"totalUnpaid": totalAmountUnpaid,
	})
}

func (a *apiPayment) countBulkPayBTC(w http.ResponseWriter, r *http.Request) {
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	count, err := a.service.CountBulkPaymentBTC(claims.Id)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}

	utils.ResponseOK(w, count)
}

func (a *apiPayment) hasReport(w http.ResponseWriter, r *http.Request) {
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	if claims.Id < 1 {
		utils.ResponseOK(w, false)
		return
	}
	hasReport := a.service.CheckHasReport(claims.Id)
	utils.ResponseOK(w, hasReport)
}

func (a *apiPayment) paymentReport(w http.ResponseWriter, r *http.Request) {
	var f portal.ReportFilter
	err := a.parseQueryAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	if claims.Id < 1 {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	payments, err := a.service.GetPaymentsForReport(claims.Id, f)
	result := make([]portal.PaymentReport, 0)
	var tmpPaymentReport = portal.PaymentReport{}
	var currentMonth = -1
	var currentYear = -1
	for index, payment := range payments {
		if currentMonth != int(payment.PaidAt.Month()) || currentYear != payment.PaidAt.Year() {
			currentMonth = int(payment.PaidAt.Month())
			currentYear = payment.PaidAt.Year()
			if !utils.IsEmpty(tmpPaymentReport.Month) {
				result = append(result, tmpPaymentReport)
			}
			tmpPaymentReport = portal.PaymentReport{}
			tmpPaymentReport.PaymentUnits = make([]portal.PaymentReportUnit, 0)
			tmpPaymentReport.Month = fmt.Sprint(currentYear, "-", currentMonth)
		}
		var paymentUnit = portal.PaymentReportUnit{}
		paymentUnit.DisplayName = payment.SenderDisplayName
		paymentUnit.Amount = payment.Amount
		paymentUnit.ExpectedAmount = payment.ExpectedAmount
		paymentUnit.PaymentMethod = payment.PaymentMethod
		tmpPaymentReport.PaymentUnits = append(tmpPaymentReport.PaymentUnits, paymentUnit)
		//if is last element
		if index == len(payments)-1 {
			result = append(result, tmpPaymentReport)
		}
	}
	utils.ResponseOK(w, result)
}

func (a *apiPayment) invoiceReport(w http.ResponseWriter, r *http.Request) {
	var f portal.ReportFilter
	err := a.parseQueryAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	if claims.Id < 1 {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	payments, err := a.service.GetForInvoiceReport(claims.Id, f)
	var reportMap = map[string][]portal.InvoiceReportUnit{}
	for _, payment := range payments {
		if len(payment.Details) == 0 {
			continue
		}
		var displayName = utils.GetUserDisplayName(payment.SenderName, payment.SenderDisplayName)
		for _, detail := range payment.Details {
			if detail.Price != 0 {
				continue
			}
			var key = displayName
			if detail.ProjectId < 1 {
				key = fmt.Sprint(key, ";")
			} else {
				key = fmt.Sprint(key, ";", detail.ProjectName)
			}
			var tmpUnit = portal.InvoiceReportUnit{}
			tmpUnit.Date = detail.Date
			tmpUnit.Description = detail.Description
			tmpUnit.Hours = detail.Quantity
			if val, ok := reportMap[key]; ok {
				val = append(val, tmpUnit)
				reportMap[key] = val
			} else {
				newUnitArr := make([]portal.InvoiceReportUnit, 0)
				newUnitArr = append(newUnitArr, tmpUnit)
				reportMap[key] = newUnitArr
			}
		}
	}
	utils.ResponseOK(w, reportMap)
}

func (a *apiPayment) getPaymentUsers(w http.ResponseWriter, r *http.Request) {
	claims, isOk := a.credentialsInfo(r)
	if !isOk || claims.Id < 1 {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("authentication failed"), nil)
		return
	}
	paymentUsers, err := a.service.GetPaymentUserList(claims.Id)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	projectsRelatedUsers, err := a.service.GetProjectRelatedMembers(claims.Id)
	if err == nil {
		for _, projectUser := range projectsRelatedUsers {
			exist := false
			for _, existUser := range paymentUsers {
				if existUser.Id == projectUser.Id {
					exist = true
					break
				}
			}
			if !exist {
				paymentUsers = append(paymentUsers, projectUser)
			}
		}
	}
	var userSelection []portal.UserSelection
	for _, user := range paymentUsers {
		userSelection = append(userSelection, portal.UserSelection{
			Id:          user.Id,
			UserName:    user.UserName,
			DisplayName: user.DisplayName,
		})
	}
	utils.ResponseOK(w, userSelection)
}

func (a *apiPayment) getExchangeList(w http.ResponseWriter, r *http.Request) {
	exchanges := a.service.ExchangeList
	if utils.IsEmpty(exchanges) {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("%s", "Get exchange list failed"), utils.ErrorBadRequest), nil)
		return
	}
	exchangeArr := strings.Split(exchanges, ",")
	resData := make([]string, 0)
	for _, exchange := range exchangeArr {
		exchange = strings.TrimSpace(exchange)
		if a.service.IsValidExchange(exchange) {
			resData = append(resData, exchange)
		}
	}
	utils.ResponseOK(w, resData)
}

func (a *apiPayment) addressReport(w http.ResponseWriter, r *http.Request) {
	var f portal.ReportFilter
	err := a.parseQueryAndValidate(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	if claims.Id < 1 {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	payments, err := a.service.GetPaymentsForReport(claims.Id, f)
	var reportMap = map[string]portal.AddressReport{}
	for _, payment := range payments {
		if payment.PaymentMethod.String() == "none" {
			continue
		}
		var displayName = utils.GetUserDisplayName(payment.SenderName, payment.SenderDisplayName)
		var tmpUnit = portal.AddressReportUnit{}
		tmpUnit.DateTime = payment.PaidAt.Format("2006/01/02")
		tmpUnit.Amount = payment.Amount
		tmpUnit.ExpectedAmount = payment.ExpectedAmount
		if val, ok := reportMap[payment.PaymentAddress]; ok {
			addressUnits := val.AddressUnits
			addressUnits = append(addressUnits, tmpUnit)
			val.AddressUnits = addressUnits
			reportMap[payment.PaymentAddress] = val
		} else {
			var tmpAddress = portal.AddressReport{}
			tmpAddress.PaymentMethod = payment.PaymentMethod.String()
			tmpAddress.DisplayName = displayName
			units := make([]portal.AddressReportUnit, 0)
			units = append(units, tmpUnit)
			tmpAddress.AddressUnits = units
			reportMap[payment.PaymentAddress] = tmpAddress
		}
	}
	utils.ResponseOK(w, reportMap)
}

func (a *apiPayment) approveRequest(w http.ResponseWriter, r *http.Request) {
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	var f portal.ApprovalRequest
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	payment, err := a.service.ApprovePaymentRequest(f.PaymentId, claims.Id)
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
		claims, isOk := a.credentialsInfo(r)
		if !isOk {
			utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
			return
		}
		if !(claims != nil && claims.Id == payment.ReceiverId) {
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

func (a *apiPayment) bulkPaidBTC(w http.ResponseWriter, r *http.Request) {
	var body portal.BulkPaidRequests
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}

	if len(body.PaymentList) == 0 {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("list payment id can't be empty or nil"), utils.ErrorBadRequest), nil)
		return
	}
	claims, isOk := a.credentialsInfo(r)
	if !isOk {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Get credentials info failed"), utils.ErrorBadRequest), nil)
		return
	}
	if err := a.service.BulkPaidBTC(claims.Id, body.TxId, body.PaymentList); err != nil {
		utils.Response(w, http.StatusForbidden, utils.InternalError.With(err), nil)
		return
	}

	utils.ResponseOK(w, nil)
}
