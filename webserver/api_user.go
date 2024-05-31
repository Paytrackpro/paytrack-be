package webserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-chi/chi/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type apiUser struct {
	*WebServer
}

func (a *apiUser) info(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.service.GetUserInfo(claims.Id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, user)
	}
}

func (a *apiUser) hidePaid(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}
	user.HidePaid = !user.HidePaid
	err = a.db.UpdateUser(user)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, Map{})
}

func (a *apiUser) changePassword(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}
	var f portal.ChangePasswordRequest
	if err := a.parseJSONAndValidate(r, &f); err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	if user.Otp && !totp.Validate(f.Otp, user.Secret) {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("failed in totp verification"), nil)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(f.OldPassword))
	if err != nil {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("your old password is not matched"), nil)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(f.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("failed when trying to encrypt password"), nil)
		return
	}
	user.PasswordHash = string(hash)
	err = a.db.UpdateUser(user)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, Map{})
}

func (a *apiUser) infoWithId(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := a.db.QueryUser(storage.UserFieldId, id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, user)
	}
}

func (a *apiUser) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	var body portal.UpdateUserRequest
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	user, err := a.service.UpdateUserInfo(uint64(body.UserId), body, true)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, user)
}

func (a *apiUser) update(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)

	var body portal.UpdateUserRequest
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	user, err := a.service.UpdateUserInfo(claims.Id, body, false)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, user)
}

func (a *apiUser) getAdminReportSummary(w http.ResponseWriter, r *http.Request) {
	var rf storage.AdminReportFilter
	if err := a.parseQueryAndValidate(r, &rf); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	payments, err := a.service.GetAllPayments(rf)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	reportSummary := portal.AdminSummaryReport{
		TotalInvoices: len(payments),
	}
	totalAmount := float64(0)
	sentInfo := portal.PaymentStatusSummary{}
	pendingInfo := portal.PaymentStatusSummary{}
	paidInfo := portal.PaymentStatusSummary{}
	usersSummaryMap := make(map[uint64]*portal.UserUsageSummary)
	userIds := make([]uint64, 0)
	for _, payment := range payments {
		if !CheckExistOnIntArray(userIds, payment.SenderId) {
			userIds = append(userIds, payment.SenderId)
		}
		if !CheckExistOnIntArray(userIds, payment.ReceiverId) {
			userIds = append(userIds, payment.ReceiverId)
		}
		totalAmount += payment.Amount
		switch payment.Status {
		case storage.PaymentStatusConfirmed:
			pendingInfo.InvoiceNum++
			pendingInfo.Amount += payment.Amount
		case storage.PaymentStatusPaid:
			paidInfo.InvoiceNum++
			paidInfo.Amount += payment.Amount
		default:
			sentInfo.InvoiceNum++
			sentInfo.Amount += payment.Amount
		}
		senderId := payment.SenderId
		receiverId := payment.ReceiverId
		var senderInMap *portal.UserUsageSummary
		var receiverInMap *portal.UserUsageSummary
		senderInMap = usersSummaryMap[senderId]
		receiverInMap = usersSummaryMap[receiverId]
		if senderInMap != nil {
			senderInMap.SendNum++
			senderInMap.SentUsd = payment.Amount
		} else {
			senderInMap = &portal.UserUsageSummary{
				Username: payment.SenderName,
				SendNum:  1,
				SentUsd:  payment.Amount,
			}
		}

		if receiverInMap != nil {
			receiverInMap.ReceiveNum++
			receiverInMap.ReceiveUsd = payment.Amount
		} else {
			receiverInMap = &portal.UserUsageSummary{
				Username:   payment.ReceiverName,
				ReceiveNum: 1,
				ReceiveUsd: payment.Amount,
			}
		}
		usersSummaryMap[senderId] = senderInMap
		usersSummaryMap[receiverId] = receiverInMap
	}
	userUsageArr := make([]portal.UserUsageSummary, 0)
	pageNum := rf.Sort.Page
	numPerpage := rf.Sort.Size
	startIndex := (pageNum - 1) * numPerpage
	endIndex := int(0)
	reportSummary.TotalAmount = totalAmount
	reportSummary.PaidInvoices = paidInfo
	reportSummary.PayableInvoices = pendingInfo
	reportSummary.SentInvoices = sentInfo
	if startIndex > len(userIds)-1 {
		reportSummary.UserUsageSummary = userUsageArr
		utils.ResponseOK(w, Map{
			"report": reportSummary,
			"count":  len(userIds),
		})
		return
	}
	if startIndex+numPerpage >= len(userIds) {
		endIndex = len(userIds) - 1
	} else {
		endIndex = startIndex + numPerpage - 1
	}

	for i := startIndex; i <= endIndex; i++ {
		userId := userIds[i]
		userUsage := usersSummaryMap[userId]
		if userUsage == nil {
			continue
		}
		userUsageArr = append(userUsageArr, *userUsage)
	}
	reportSummary.UserUsageSummary = userUsageArr
	utils.ResponseOK(w, Map{
		"report": reportSummary,
		"count":  len(userIds),
	})
}

func CheckExistOnIntArray(intArr []uint64, checkInt uint64) bool {
	for _, num := range intArr {
		if num == checkInt {
			return true
		}
	}
	return false
}

func (a *apiUser) getListUsers(w http.ResponseWriter, r *http.Request) {
	var f storage.UserFilter
	if err := a.parseQueryAndValidate(r, &f); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	//default sortable is lastSeen desc (newest before)
	if utils.IsEmpty(f.Sort.Order) {
		f.Sort.Order = "lastSeen desc"
	}
	var users []storage.User
	if err := a.db.GetList(&f, &users); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	count, _ := a.db.Count(&f, &storage.User{})
	utils.ResponseOK(w, Map{
		"users": users,
		"count": count,
	})
}

func (a *apiUser) getUserSelectionList(w http.ResponseWriter, r *http.Request) {
	var f storage.UserFilter
	if err := a.parseQueryAndValidate(r, &f); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	var users []storage.User
	if err := a.db.GetList(&f, &users); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	var userSelection []portal.UserSelection
	for _, user := range users {
		userSelection = append(userSelection, portal.UserSelection{
			Id:          user.Id,
			UserName:    user.UserName,
			DisplayName: user.DisplayName,
		})
	}
	utils.ResponseOK(w, userSelection)
}

func (a *apiUser) checkingUserExist(w http.ResponseWriter, r *http.Request) {
	userName := r.FormValue("userName")
	claims, _ := a.credentialsInfo(r)
	if claims.UserName == userName {
		utils.ResponseOK(w, Map{
			"found":   false,
			"message": "userName must not be yours",
		})
		return
	}
	user, err := a.db.QueryUser(storage.UserFieldUName, userName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ResponseOK(w, Map{
				"found":   false,
				"message": "userName not found",
			})
		} else {
			utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		}
		return
	}
	utils.ResponseOK(w, Map{
		"found":           true,
		"id":              user.Id,
		"userName":        user.UserName,
		"paymentSettings": user.PaymentSettings,
	})
}

func (a *apiUser) usersExist(w http.ResponseWriter, r *http.Request) {
	userName := r.FormValue("userNames")
	claims, _ := a.credentialsInfo(r)
	if utils.IsEmpty(userName) {
		utils.Response(w, http.StatusBadRequest, fmt.Errorf("userNames is null or empty"), nil)
		return
	}

	listUserName := strings.Split(userName, ",")
	for _, v := range listUserName {
		if v == claims.UserName {
			utils.Response(w, http.StatusBadRequest, fmt.Errorf("userName must not be yours"), nil)
			return
		}
	}

	users, err := a.db.QueryUserWithList("user_name", listUserName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
			return
		} else {
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}
	}

	if len(users) == len(listUserName) {
		utils.ResponseOK(w, users)
		return
	} else {
		userMap := make(map[string]bool)
		for _, u := range users {
			userMap[u.UserName] = true
		}

		for _, v := range listUserName {
			if !userMap[v] {
				utils.Response(w, http.StatusBadRequest, fmt.Errorf("user %s not found", v), nil)
				return
			}
		}
	}
}

func (a *apiUser) membersExist(w http.ResponseWriter, r *http.Request) {
	userNames := r.FormValue("userNames")
	if utils.IsEmpty(userNames) {
		return
	}
	listUserName := strings.Split(userNames, ",")
	users, err := a.db.QueryUserWithList("user_name", listUserName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
			return
		} else {
			utils.Response(w, http.StatusInternalServerError, err, nil)
			return
		}
	}

	if len(users) == len(listUserName) {
		utils.ResponseOK(w, users)
		return
	} else {
		userMap := make(map[string]bool)
		for _, u := range users {
			userMap[u.UserName] = true
		}

		for _, v := range listUserName {
			if !userMap[v] {
				utils.Response(w, http.StatusBadRequest, fmt.Errorf("Member '%s' not found", v), nil)
				return
			}
		}
	}
}

func (a *apiUser) generateQr(w http.ResponseWriter, r *http.Request) {
	var f portal.GenerateQRForm
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldUName, claims.UserName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Response(w, http.StatusNotFound, utils.InvalidCredential, nil)
			return
		}

		utils.Response(w, http.StatusInternalServerError, err, nil)

		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(f.Password))
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.InvalidCredential, nil)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "MGMT",
		AccountName: user.UserName,
	})
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	qrImage, err := key.Image(200, 200)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	imgBase64Str, err := utils.ImageToBase64(qrImage)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.SetValue(&user.Secret, key.Secret())
	err = a.db.UpdateUser(user)

	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, Map{
		"mfa_qr_image": imgBase64Str,
		"secret_key":   key.Secret(),
		"account":      key.AccountName(),
		"time_based":   true,
	})
}

func (a *apiUser) disableOtp(w http.ResponseWriter, r *http.Request) {
	var f portal.OtpForm
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldUName, claims.UserName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Response(w, http.StatusNotFound, utils.InvalidCredential, nil)
			return
		}
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(f.Password))
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.InvalidCredential, nil)
		return
	}

	verified := totp.Validate(f.Otp, user.Secret)

	if !verified {
		err := utils.NewError(fmt.Errorf("OTP is not valid"), utils.ErrorObjectExist)
		utils.Response(w, http.StatusBadRequest, err, nil)

		return
	}

	utils.SetValue(&user.Otp, false)
	err = a.db.UpdateUser(user)

	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, Map{})
}

func (a *apiUser) updatePaymentSetting(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	var body portal.ListPaymentSettingRequest
	err := a.parseJSON(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	// validate list approvers
	for _, approver := range body.List {
		if approver.SendUserId == claims.Id {
			e := fmt.Errorf("the sender can't be you")
			utils.Response(w, http.StatusBadRequest, e, nil)
			return
		}

		for _, approverId := range approver.ApproverIds {
			if approverId == claims.Id {
				e := fmt.Errorf("the approver can't be you")
				utils.Response(w, http.StatusBadRequest, e, nil)
				return
			}
		}
	}

	approverSetting, err := a.service.UpdateApproverSetting(claims.Id, body.List)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, approverSetting)
}

func (a *apiUser) getPaymentSetting(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	app := portal.Approvers{}
	app.Id = claims.Id
	var approvers []storage.ApproverSettings
	if err := a.db.GetList(&app, &approvers); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	temMap := make(map[string][]storage.ApproverSettings, 0)
	for _, appr := range approvers {
		temMap[appr.SendUserName] = append(temMap[appr.SendUserName], appr)
	}

	res := make([]map[string]interface{}, 0)
	for _, v := range temMap {
		approvers := make([]map[string]interface{}, 0)
		var showCost = false
		for _, appro := range v {
			approvers = append(approvers, Map{
				"approverName": appro.ApproverName,
				"approverId":   appro.ApproverId,
			})
			showCost = appro.ShowCost
		}

		res = append(res, Map{
			"sendUserId":   v[0].SendUserId,
			"sendUserName": v[0].SendUserName,
			"recipientId":  v[0].RecipientId,
			"showCost":     showCost,
			"approvers":    approvers,
		})
	}

	utils.ResponseOK(w, res)
}
