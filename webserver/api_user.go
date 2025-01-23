package webserver

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

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

func (a *apiUser) showApproved(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	user, err := a.db.QueryUser(storage.UserFieldId, claims.Id)
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
		return
	}
	user.ShowApproved = !user.ShowApproved
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

func (a *apiUser) resumeTimer(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	//check if exist timer is running
	runningTimer, runningErr := a.service.GetRunningTimer(claims.Id)
	if runningErr != nil || runningTimer == nil {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf("%s", "Get running timer error"), nil)
		return
	}

	//check pausing status
	if runningTimer.Fininshed || !runningTimer.Pausing {
		utils.ResponseOK(w, Map{
			"error": true,
			"msg":   "Timer has Finished or is not in pause state. Can not resume",
		})
		return
	}
	//check newest sum
	pauseState := runningTimer.PauseState
	if len(pauseState) < 1 {
		utils.ResponseOK(w, Map{
			"error": true,
			"msg":   "Timer has not been paused. Cannot resume",
		})
		return
	}

	var lastIndex int
	var lastPause *storage.PauseStatus
	for index, pauseStatus := range pauseState {
		if !pauseStatus.Stop.IsZero() {
			continue
		}
		startPause := pauseStatus.Start
		if lastPause == nil {
			lastPause = &pauseStatus
			lastIndex = index
			continue
		}
		if startPause.After(lastPause.Start) {
			lastIndex = index
			lastPause = &pauseStatus
		}
	}

	if lastPause == nil {
		utils.ResponseOK(w, Map{
			"error": true,
			"msg":   "Timer has not been paused. Cannot resume",
		})
		return
	}

	lastPause.Stop = time.Now()
	runningTimer.PauseState[lastIndex] = *lastPause
	runningTimer.Pausing = false
	//update running timer
	if err := a.db.UpdateUserTimer(runningTimer); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	socketData := storage.UserTimerSockerData{
		UserId:  claims.Id,
		Working: true,
		Pausing: false,
	}
	a.HandlerForReloadAdminUsers(socketData)

	utils.ResponseOK(w, Map{
		"error":        false,
		"runningTimer": runningTimer,
	})
}

func (a *apiUser) deleteTimer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a.db.GetDB().Where("id = ?", id).Delete(&storage.UserTimer{})
	utils.ResponseOK(w, nil)
}

func (a *apiUser) updateTimer(w http.ResponseWriter, r *http.Request) {
	var body portal.TimerUpdateRequest
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}
	//get timer with id
	userTimer, err := a.service.GetUserTimer(body.TimerId)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf("%s", "Get user timer error"), nil)
		return
	}
	if body.ProjectId >= 0 {
		userTimer.ProjectId = uint64(body.ProjectId)
	}
	if !utils.IsEmpty(body.Description) {
		userTimer.Description = body.Description
	}
	//update running timer
	if err := a.db.UpdateUserTimer(&userTimer); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, userTimer)
}

func (a *apiUser) stopTimer(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	//check if exist timer is running
	runningTimer, runningErr := a.service.GetRunningTimer(claims.Id)
	if runningErr != nil || runningTimer == nil {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf("%s", "Get running timer error"), nil)
		return
	}
	//check pausing status
	if runningTimer.Fininshed {
		utils.ResponseOK(w, Map{
			"error": true,
			"msg":   "Timer has finished, cannot be stopped!",
		})
		return
	}

	//update running timer
	runningTimer.Stop = time.Now()
	var pauseIndex int
	var updatePauseStatus *storage.PauseStatus
	if runningTimer.Pausing && len(runningTimer.PauseState) > 0 {
		for index, pauseStatus := range runningTimer.PauseState {
			if !pauseStatus.Stop.IsZero() {
				continue
			}
			pauseStatus.Stop = time.Now()
			updatePauseStatus = &pauseStatus
			pauseIndex = index
			break
		}
		if updatePauseStatus != nil {
			runningTimer.PauseState[pauseIndex] = *updatePauseStatus
		}
	}

	totalSecond := uint64(0)
	allDurationSec := utils.GetSecondDurationFromStartEnd(runningTimer.Start, runningTimer.Stop)
	totalPauseSec := uint64(0)
	//caculate sum of pausing seconds
	if len(runningTimer.PauseState) > 0 {
		for _, pauseState := range runningTimer.PauseState {
			if pauseState.Start.IsZero() || pauseState.Stop.IsZero() {
				continue
			}
			pauseSec := utils.GetSecondDurationFromStartEnd(pauseState.Start, pauseState.Stop)
			totalPauseSec += pauseSec
		}
	}
	totalSecond = allDurationSec - totalPauseSec
	runningTimer.Duration = totalSecond
	runningTimer.Fininshed = true
	runningTimer.Pausing = false

	//update running timer
	if err := a.db.UpdateUserTimer(runningTimer); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	socketData := storage.UserTimerSockerData{
		UserId:  claims.Id,
		Working: false,
		Pausing: false,
	}
	a.HandlerForReloadAdminUsers(socketData)

	utils.ResponseOK(w, Map{
		"error":        false,
		"runningTimer": runningTimer,
	})
}

func (a *apiUser) pauseTimer(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	//check if exist timer is running
	runningTimer, runningErr := a.service.GetRunningTimer(claims.Id)
	if runningErr != nil || runningTimer == nil {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf("%s", "Get running timer error"), nil)
		return
	}

	//check pausing status
	if runningTimer.Fininshed || runningTimer.Pausing {
		utils.ResponseOK(w, Map{
			"error": true,
			"msg":   "Timer has been paused or finished. Can not pause",
		})
		return
	}
	//create new pausing state
	newPausingState := storage.PauseStatus{
		Start: time.Now(),
	}
	if len(runningTimer.PauseState) > 0 {
		runningTimer.PauseState = append(runningTimer.PauseState, newPausingState)
	} else {
		runningTimer.PauseState = []storage.PauseStatus{newPausingState}
	}

	runningTimer.Pausing = true
	//update running timer
	if err := a.db.UpdateUserTimer(runningTimer); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	socketData := storage.UserTimerSockerData{
		UserId:  claims.Id,
		Working: true,
		Pausing: true,
	}
	a.HandlerForReloadAdminUsers(socketData)

	utils.ResponseOK(w, Map{
		"error":        false,
		"runningTimer": runningTimer,
	})
}

func (a *apiUser) startTimer(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	//check if exist timer is running
	runningTimer, runningErr := a.service.GetRunningTimer(claims.Id)
	if runningErr != nil {
		utils.Response(w, http.StatusInternalServerError, runningErr, nil)
		return
	}

	//if exist running timer, return error running timer
	if runningTimer != nil {
		utils.Response(w, http.StatusInternalServerError, fmt.Errorf("%s", "Other timer is running. Can't start new timer"), nil)
		return
	}

	//Create new timer
	var userTimer = storage.UserTimer{
		UserId: claims.Id,
		Start:  time.Now(),
	}
	if err := a.db.CreateUserTimer(&userTimer); err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	socketData := storage.UserTimerSockerData{
		UserId:  claims.Id,
		Working: true,
		Pausing: false,
	}
	a.HandlerForReloadAdminUsers(socketData)
	utils.ResponseOK(w, Map{
		"runningTimer": userTimer,
	})
}

func (a *apiUser) HandlerForReloadAdminUsers(socketData storage.UserTimerSockerData) {
	if adminIds, err := a.service.GetAdminIds(); err == nil {
		adminIdStrs := make([]string, 0)
		for _, adminId := range adminIds {
			adminIdStrs = append(adminIdStrs, fmt.Sprint(adminId))
		}
		if len(adminIdStrs) > 0 {
			a.reloadUserList(adminIdStrs, socketData)
		}
	}
}

func (a *apiUser) getTimeLogList(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	var filter storage.AdminReportFilter
	err := a.parseQueryAndValidate(r, &filter)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}

	timerList, err := a.service.GetLogTimeList(claims.Id, filter)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	timerCount, countErr := a.service.CountLogTimer(claims.Id, filter)
	if countErr != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(countErr, utils.ErrorInternalCode), nil)
		return
	}
	utils.ResponseOK(w, Map{
		"timers": timerList,
		"count":  timerCount,
	})
}

func (a *apiUser) getRunningTimer(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	//check if exist timer is running
	runningTimer, runningErr := a.service.GetRunningTimer(claims.Id)
	if runningErr != nil || runningTimer == nil {
		utils.ResponseOK(w, Map{
			"exist": false,
		})
		return
	}

	totalSecond := uint64(0)
	startDate := runningTimer.Start
	var endDate time.Time
	if runningTimer.Fininshed {
		endDate = runningTimer.Stop
	} else {
		endDate = time.Now()
	}

	allDurationSec := utils.GetSecondDurationFromStartEnd(startDate, endDate)
	totalPauseSec := uint64(0)
	//caculate sum of pausing seconds
	if len(runningTimer.PauseState) > 0 {
		for _, pauseState := range runningTimer.PauseState {
			if pauseState.Start.IsZero() {
				continue
			}
			var pauseStopTime time.Time
			if pauseState.Stop.IsZero() {
				pauseStopTime = time.Now()
			} else {
				pauseStopTime = pauseState.Stop
			}
			pauseSec := utils.GetSecondDurationFromStartEnd(pauseState.Start, pauseStopTime)
			totalPauseSec += pauseSec
		}
	}
	totalSecond = allDurationSec - totalPauseSec

	utils.ResponseOK(w, Map{
		"runningTimer": runningTimer,
		"totalSeconds": totalSecond,
		"exist":        true,
	})
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
		if strings.Contains(payment.SenderName, rf.UserName) {
			if !CheckExistOnIntArray(userIds, payment.SenderId) {
				userIds = append(userIds, payment.SenderId)
			}
		}
		if strings.Contains(payment.ReceiverName, rf.UserName) {
			if !CheckExistOnIntArray(userIds, payment.ReceiverId) {
				userIds = append(userIds, payment.ReceiverId)
			}
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
			senderInMap.SentUsd += payment.Amount
		} else {
			senderInMap = &portal.UserUsageSummary{
				Username: payment.SenderName,
				SendNum:  1,
				SentUsd:  payment.Amount,
			}
		}
		if receiverInMap != nil {
			receiverInMap.ReceiveNum++
			receiverInMap.ReceiveUsd += payment.Amount
		} else {
			receiverInMap = &portal.UserUsageSummary{
				Username:   payment.ReceiverName,
				ReceiveNum: 1,
				ReceiveUsd: payment.Amount,
				PaidNum:    0,
				PaidUsd:    0,
			}
		}
		if payment.Status == storage.PaymentStatusPaid {
			if rf.UserName == "" {
				senderInMap.GotPaidNum++
				senderInMap.GotPaidUsd += payment.Amount
				receiverInMap.PaidNum++
				receiverInMap.PaidUsd += payment.Amount
			} else {
				isReceiverSearched := strings.Contains(payment.ReceiverName, rf.UserName)
				if isReceiverSearched {
					receiverInMap.PaidNum++
					receiverInMap.PaidUsd += payment.Amount
					senderInMap.GotPaidNum++
					senderInMap.GotPaidUsd += payment.Amount
				} else {
					receiverInMap.PaidNum++
					receiverInMap.PaidUsd += payment.Amount
					senderInMap.GotPaidNum++
					senderInMap.GotPaidUsd += payment.Amount
				}
			}
		}
		usersSummaryMap[senderId] = senderInMap
		usersSummaryMap[receiverId] = receiverInMap
	}
	userUsageArr := make([]portal.UserUsageSummary, 0)
	var pageNum, numPerpage, startIndex, endIndex int
	if rf.Sort.Size == 0 {
		pageNum = 1
		numPerpage = len(userIds)
		startIndex = 0
		endIndex = len(userIds) - 1
	} else {
		pageNum = rf.Sort.Page
		numPerpage = rf.Sort.Size
		startIndex = (pageNum - 1) * numPerpage
		endIndex = int(0)
	}

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
		if userUsage == nil || userUsage.Username == "" {
			continue
		}
		userUsageArr = append(userUsageArr, *userUsage)
	}

	if strings.Contains(rf.Sort.Order, "username") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].Username > userUsageArr[b].Username
			} else {
				return userUsageArr[a].Username < userUsageArr[b].Username
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "send") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].SendNum > userUsageArr[b].SendNum
			} else {
				return userUsageArr[a].SendNum < userUsageArr[b].SendNum
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "sendusd") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].SentUsd > userUsageArr[b].SentUsd
			} else {
				return userUsageArr[a].SentUsd < userUsageArr[b].SentUsd
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "receiveusd") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].ReceiveUsd > userUsageArr[b].ReceiveUsd
			} else {
				return userUsageArr[a].ReceiveUsd < userUsageArr[b].ReceiveUsd
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "receive") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].ReceiveNum > userUsageArr[b].ReceiveNum
			} else {
				return userUsageArr[a].ReceiveNum < userUsageArr[b].ReceiveNum
			}
		})
	}
	if strings.Contains(rf.Sort.Order, "paidusd") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].PaidUsd > userUsageArr[b].PaidUsd
			} else {
				return userUsageArr[a].PaidUsd < userUsageArr[b].PaidUsd
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "paid") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].PaidNum > userUsageArr[b].PaidNum
			} else {
				return userUsageArr[a].PaidNum < userUsageArr[b].PaidNum
			}
		})
	}
	reportSummary.UserUsageSummary = userUsageArr
	utils.ResponseOK(w, Map{
		"report": reportSummary,
		"count":  len(userIds),
	})
}

func (a *apiUser) getAdminReportSummaryBanGoc(w http.ResponseWriter, r *http.Request) {
	var rf storage.AdminReportFilter
	if err := a.parseQueryAndValidate(r, &rf); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	payments, err := a.service.GetAllPaymentsBanGoc(rf)
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
				PaidNum:    0,
				PaidUsd:    0,
			}
		}
		if payment.Status == storage.PaymentStatusPaid {
			receiverInMap.PaidNum++
			receiverInMap.PaidUsd = payment.Amount
		}
		usersSummaryMap[senderId] = senderInMap
		usersSummaryMap[receiverId] = receiverInMap
	}
	userUsageArr := make([]portal.UserUsageSummary, 0)
	var pageNum, numPerpage, startIndex, endIndex int
	if rf.Sort.Size == 0 {
		pageNum = 1
		numPerpage = len(userIds)
		startIndex = 0
		endIndex = len(userIds) - 1
	} else {
		pageNum = rf.Sort.Page
		numPerpage = rf.Sort.Size
		startIndex = (pageNum - 1) * numPerpage
		endIndex = int(0)
	}

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
		if userUsage == nil || userUsage.Username == "" {
			continue
		}
		userUsageArr = append(userUsageArr, *userUsage)
	}
	if strings.Contains(rf.Sort.Order, "username") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].Username > userUsageArr[b].Username
			} else {
				return userUsageArr[a].Username < userUsageArr[b].Username
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "send") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].SendNum > userUsageArr[b].SendNum
			} else {
				return userUsageArr[a].SendNum < userUsageArr[b].SendNum
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "sendusd") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].SentUsd > userUsageArr[b].SentUsd
			} else {
				return userUsageArr[a].SentUsd < userUsageArr[b].SentUsd
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "receiveusd") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].ReceiveUsd > userUsageArr[b].ReceiveUsd
			} else {
				return userUsageArr[a].ReceiveUsd < userUsageArr[b].ReceiveUsd
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "receive") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].ReceiveNum > userUsageArr[b].ReceiveNum
			} else {
				return userUsageArr[a].ReceiveNum < userUsageArr[b].ReceiveNum
			}
		})
	}
	if strings.Contains(rf.Sort.Order, "paidusd") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].PaidUsd > userUsageArr[b].PaidUsd
			} else {
				return userUsageArr[a].PaidUsd < userUsageArr[b].PaidUsd
			}
		})
	}

	if strings.Contains(rf.Sort.Order, "paid") {
		sort.Slice(userUsageArr, func(a, b int) bool {
			if strings.Contains(rf.Sort.Order, "desc") {
				return userUsageArr[a].PaidNum > userUsageArr[b].PaidNum
			} else {
				return userUsageArr[a].PaidNum < userUsageArr[b].PaidNum
			}
		})
	}
	reportSummary.UserUsageSummary = userUsageArr
	utils.ResponseOK(w, Map{
		"report": reportSummary,
		"count":  len(userIds),
	})
}

func (a *apiUser) getAdminReportSummaryUserDetail(w http.ResponseWriter, r *http.Request) {
	var rf storage.AdminReportFilterUserDetail
	if err := a.parseQueryAndValidate(r, &rf); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}

	payments, err := a.service.GetAllPaymentsForReportUserDetail(rf)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}

	reportSummary := portal.AdminSummaryReportDetailUser{
		TotalInvoices: len(payments),
	}

	totalAmount := float64(0)
	sentInfo := portal.PaymentStatusSummary{}
	pendingInfo := portal.PaymentStatusSummary{}
	paidInfo := portal.PaymentStatusSummary{}
	userDetailUsageArr := make([]portal.UserDetailUsageSummary, 0)

	for _, payment := range payments {
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

		detail := portal.UserDetailUsageSummary{
			Sender:       payment.SenderName,
			Receiver:     payment.ReceiverName,
			Status:       int(payment.Status),
			Amount:       payment.Amount,
			AcceptedCoin: payment.PaymentMethod.String(),
			StartDate:    payment.StartDate,
			LastEdited:   payment.UpdatedAt,
		}
		userDetailUsageArr = append(userDetailUsageArr, detail)
	}

	sortOrder := strings.TrimSpace(strings.ToLower(rf.Sort.Order))
	if sortOrder != "" {
		parts := strings.Fields(sortOrder)
		sortField := ""
		sortDesc := false

		if len(parts) >= 1 {
			sortField = parts[0]
		}
		if len(parts) >= 2 {
			sortDesc = parts[1] == "desc"
		}

		sort.Slice(userDetailUsageArr, func(i, j int) bool {
			var result bool
			switch sortField {
			case "sender":
				result = strings.ToLower(userDetailUsageArr[i].Sender) < strings.ToLower(userDetailUsageArr[j].Sender)
			case "receiver":
				result = strings.ToLower(userDetailUsageArr[i].Receiver) < strings.ToLower(userDetailUsageArr[j].Receiver)
			case "amount":
				result = userDetailUsageArr[i].Amount < userDetailUsageArr[j].Amount
			case "startdate":
				result = userDetailUsageArr[i].StartDate.Before(userDetailUsageArr[j].StartDate)
			case "lastedited":
				result = userDetailUsageArr[i].LastEdited.Before(userDetailUsageArr[j].LastEdited)
			default:
				result = userDetailUsageArr[i].StartDate.Before(userDetailUsageArr[j].StartDate)
			}
			if sortDesc {
				return !result
			}
			return result
		})
	}

	totalItems := len(userDetailUsageArr)
	pageNum := 1
	numPerPage := totalItems

	if rf.Sort.Size > 0 {
		pageNum = rf.Sort.Page
		numPerPage = rf.Sort.Size
	}

	startIndex := (pageNum - 1) * numPerPage
	endIndex := startIndex + numPerPage

	if startIndex >= totalItems {
		userDetailUsageArr = []portal.UserDetailUsageSummary{}
	} else {
		if endIndex > totalItems {
			endIndex = totalItems
		}
		userDetailUsageArr = userDetailUsageArr[startIndex:endIndex]
	}

	reportSummary.TotalAmount = totalAmount
	reportSummary.PaidInvoices = paidInfo
	reportSummary.PayableInvoices = pendingInfo
	reportSummary.SentInvoices = sentInfo
	reportSummary.UserDetailUsageSummary = userDetailUsageArr

	utils.ResponseOK(w, Map{
		"report": reportSummary,
		"count":  totalItems,
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

func (a *apiUser) reloadUserList(rooms []string, data interface{}) {
	for _, room := range rooms {
		log.Debug("send data ", data, " to room ", room)
		a.socket.BroadcastToRoom("", room, "reloadUserList", data)
	}
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

	count, _ := a.db.Count(&f, &storage.User{})
	if f.Size == 0 {
		f.Size = int(count)
		f.Page = 1
	}
	var users []storage.User
	if err := a.db.GetList(&f, &users); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	//get working flg
	workingMap, err := a.service.GetWorkingUserList()
	userResList := make([]storage.UserWorkingDisplay, 0)
	for _, user := range users {
		working := false
		pausing := false
		if err == nil {
			tmpPausing, exist := workingMap[user.Id]
			if exist {
				working = true
				pausing = tmpPausing
			}
		}
		userRes := storage.UserWorkingDisplay{
			User:    user,
			Working: working,
			Pausing: pausing,
		}
		userResList = append(userResList, userRes)
	}

	utils.ResponseOK(w, Map{
		"users": userResList,
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
func (a *apiUser) getUserSenderPaid(w http.ResponseWriter, r *http.Request) {
	claims, _ := a.credentialsInfo(r)
	userCurrent, err := a.service.GetUserInfo(claims.Id)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	var f storage.UserFilter
	if err := a.parseQueryAndValidate(r, &f); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}

	var payments []storage.Payment
	if err := a.db.GetUserSender(&f, &payments); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}

	uniqueSenders := make(map[uint64]portal.UserSelection)
	for _, p := range payments {
		if _, exists := uniqueSenders[p.SenderId]; exists {
			continue
		}
		if p.ReceiverId == userCurrent.Id {
			user, err := a.service.GetUserInfo(p.SenderId)
			if err != nil {
				utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
				return
			}
			uniqueSenders[p.SenderId] = portal.UserSelection{
				Id:          user.Id,
				UserName:    user.UserName,
				DisplayName: user.DisplayName,
			}
		}
	}

	var userSelection []portal.UserSelection
	for _, user := range uniqueSenders {
		userSelection = append(userSelection, user)
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

func (a *apiUser) checkingProjectMemberExist(w http.ResponseWriter, r *http.Request) {
	userName := r.FormValue("userName")
	user, err := a.db.QueryUser(storage.UserFieldUName, userName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ResponseOK(w, Map{
				"found":   false,
				"message": "user not found",
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
		"displayName":     user.DisplayName,
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
