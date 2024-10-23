package service

import (
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"gorm.io/gorm"
)

func (s *Service) GetBulkPaymentBTC(userId uint64, page, pageSize int, order string) ([]storage.Payment, int64, error) {
	if page != 0 {
		page = page - 1
	}
	var count int64
	payments := make([]storage.Payment, 0)
	//Get count of payments
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM payments WHERE payment_settings @> '[{"type": "%s"}]' AND status <> %d AND status <> %d AND status <> %d AND receiver_id = %d`,
		utils.PaymentTypeBTC.String(), storage.PaymentStatusPaid, storage.PaymentStatusRejected, storage.PaymentStatusCreated, userId)
	if err := s.db.Raw(countQuery).Scan(&count).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, 0, nil
		}
		return nil, 0, err
	}

	if pageSize == 0 {
		pageSize = int(count)
		page = 1
	}

	offset := page * pageSize
	query := fmt.Sprintf(`SELECT * FROM payments WHERE payment_settings @> '[{"type": "%s"}]' AND status <> %d AND status <> %d AND status <> %d AND receiver_id = %d LIMIT %d OFFSET %d`,
		utils.PaymentTypeBTC.String(), storage.PaymentStatusPaid, storage.PaymentStatusRejected, storage.PaymentStatusCreated, userId, pageSize, offset)

	if err := s.db.Raw(query).Scan(&payments).Order(order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, 0, nil
		}
		return nil, 0, err
	}

	return payments, count, nil
}

func (s *Service) CountBulkPaymentBTC(userId uint64) (int64, error) {
	var count int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM payments WHERE payment_settings @> '[{"type": "%s"}]'  AND status <> %d AND status <> %d AND status <> %d AND receiver_id = %d`, utils.PaymentTypeBTC.String(), storage.PaymentStatusPaid, storage.PaymentStatusRejected, storage.PaymentStatusCreated, userId)

	if err := s.db.Raw(countQuery).Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) GetRequestSummary(userId uint64, summaryFilter portal.SummaryFilter) (portal.PaymentSummary, error) {
	var requestSentCount int64
	var requestReceivedCount int64
	var requestPaidCount int64
	var totalPaid sql.NullFloat64
	var totalReceived sql.NullFloat64
	var paymentSummary portal.PaymentSummary
	var idArray = strings.Split(summaryFilter.Ids, ",")
	var idsInt = make([]int, len(idArray))
	for i, v := range idArray {
		idsInt[i], _ = strconv.Atoi(v)
	}
	if len(summaryFilter.Ids) == 0 {
		buildRequestSentCount := s.db.Model(&storage.Payment{}).Where("sender_id = ? AND status <> ? AND EXTRACT(MONTH FROM sent_at) = ?", userId, storage.PaymentStatusCreated, summaryFilter.Month)
		if err := buildRequestSentCount.Count(&requestSentCount).Error; err != nil {
			return paymentSummary, err
		}
	} else {
		buildRequestSentCount := s.db.Model(&storage.Payment{}).Where("sender_id = ? AND status <> ? AND EXTRACT(MONTH FROM sent_at) = ? AND receiver_id IN ?", userId, storage.PaymentStatusCreated, summaryFilter.Month, idsInt)
		if err := buildRequestSentCount.Count(&requestSentCount).Error; err != nil {
			return paymentSummary, err
		}
	}
	buildRequestReceivedCount := s.db.Model(&storage.Payment{}).Where("receiver_id = ? AND status <> ? AND EXTRACT(MONTH FROM sent_at) = ?", userId, storage.PaymentStatusCreated, summaryFilter.Month)
	if err := buildRequestReceivedCount.Count(&requestReceivedCount).Error; err != nil {
		return paymentSummary, err
	}

	buildRequestPaidCount := s.db.Model(&storage.Payment{}).Where("receiver_id = ? AND status = ? AND EXTRACT(MONTH FROM sent_at) = ?", userId, storage.PaymentStatusPaid, summaryFilter.Month)
	if err := buildRequestPaidCount.Count(&requestPaidCount).Error; err != nil {
		return paymentSummary, err
	}

	totalPaidQuery := fmt.Sprintf(`SELECT sum(amount) FROM payments WHERE status = %d AND receiver_id = %d AND EXTRACT(MONTH FROM paid_at) = %d`, storage.PaymentStatusPaid, userId, summaryFilter.Month)

	if err := s.db.Raw(totalPaidQuery).Scan(&totalPaid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return paymentSummary, err
		}
		return paymentSummary, err
	}
	var totalRececiverQuery = fmt.Sprintf(`SELECT sum(amount) FROM payments WHERE status = %d AND sender_id = %d AND EXTRACT(MONTH FROM paid_at) = %d`, storage.PaymentStatusPaid, userId, summaryFilter.Month)
	if len(summaryFilter.Ids) > 0 {
		totalRececiverQuery = fmt.Sprintf(`SELECT sum(amount) FROM payments WHERE status = %d AND sender_id = %d AND EXTRACT(MONTH FROM paid_at) = %d AND receiver_id IN (%s)`, storage.PaymentStatusPaid, userId, summaryFilter.Month, summaryFilter.Ids)
	}
	if err := s.db.Raw(totalRececiverQuery).Scan(&totalReceived).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return paymentSummary, err
		}
		return paymentSummary, err
	}

	paymentSummary.RequestSent = uint64(requestSentCount)
	paymentSummary.RequestReceived = uint64(requestReceivedCount)
	paymentSummary.RequestPaid = uint64(requestPaidCount)
	paymentSummary.TotalPaid = totalPaid.Float64
	paymentSummary.TotalReceived = totalReceived.Float64

	return paymentSummary, nil
}

func (s *Service) CreatePayment(userId uint64, userName string, displayName string, showDraftForRecipient bool, request portal.PaymentRequest) (*storage.Payment, error) {
	var reciver storage.User
	payment := storage.Payment{
		SenderId:              userId,
		SenderName:            userName,
		SenderDisplayName:     displayName,
		Description:           request.Description,
		Details:               request.Details,
		Status:                request.Status,
		HourlyRate:            request.HourlyRate,
		PaymentSettings:       request.PaymentSettings,
		ShowDraftRecipient:    showDraftForRecipient,
		ShowDateOnInvoiceLine: request.ShowDateOnInvoiceLine,
		ShowProjectOnInvoice:  request.ShowProjectOnInvoice,
	}

	if payment.ShowProjectOnInvoice {
		payment.ProjectId = request.ProjectId
		payment.ProjectName = request.ProjectName
	}

	// payment is internal
	if request.ContactMethod == storage.PaymentTypeInternal {
		if request.ReceiverId > 0 {
			if err := s.db.Where("id = ?", request.ReceiverId).First(&reciver).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil, utils.NewError(fmt.Errorf("receiver not found"), utils.ErrorBadRequest)
				}
				return nil, err
			}
			payment.ReceiverId = request.ReceiverId
			payment.ReceiverName = reciver.UserName
			payment.ReceiverDisplayName = reciver.DisplayName
			if len(payment.SenderDisplayName) == 0 {
				payment.SenderDisplayName = payment.SenderName
			}
			if len(payment.ReceiverDisplayName) == 0 {
				payment.ReceiverDisplayName = payment.ReceiverName
			}
		}
	} else {
		// payment is external
		payment.ExternalEmail = request.ExternalEmail
	}

	if len(request.Details) > 0 {
		amount, err := calculateAmount(request)
		if err != nil {
			return nil, utils.NewError(err, utils.ErrorBadRequest)
		}
		payment.Amount = amount
		startDate, err := getStartDate(request)
		if err != nil {
			return nil, utils.NewError(err, utils.ErrorBadRequest)
		}
		payment.StartDate = startDate
	} else {
		payment.Amount = request.Amount
		payment.StartDate = time.Now()
	}

	if payment.Status == storage.PaymentStatusSent {
		//if status is sent, set sentAt is now
		payment.SentAt = time.Now()
		//get project ids on payment
		projectIds := make([]string, 0)
		if len(payment.Details) > 0 {
			for _, detail := range payment.Details {
				if detail.ProjectId < 1 {
					continue
				}
				projectIdStr := fmt.Sprintf("%d", detail.ProjectId)
				if !slices.Contains(projectIds, projectIdStr) {
					projectIds = append(projectIds, projectIdStr)
				}
			}
		}

		projects, err := s.GetPaymentProjects(projectIds)
		if err != nil {
			return nil, err
		}
		var approversList []storage.Member
		for _, project := range projects {
			for _, approver := range project.Approvers {
				exist := false
				for _, approveUser := range approversList {
					if approveUser.MemberId == approver.MemberId {
						exist = true
						break
					}
				}
				if !exist {
					approversList = append(approversList, approver)
				}
			}
		}

		if len(approversList) > 0 {
			approvers := storage.Approvers{}
			for _, approver := range approversList {
				isApproved := false
				if approver.MemberId == payment.ReceiverId || approver.MemberId == payment.SenderId {
					isApproved = true
				}
				tempApprover := storage.Approver{
					ApproverId: approver.MemberId,
					IsApproved: isApproved,
					ShowCost:   true,
				}
				if utils.IsEmpty(approver.DisplayName) {
					tempApprover.ApproverName = approver.UserName
				} else {
					tempApprover.ApproverName = approver.DisplayName
				}
				approvers = append(approvers, tempApprover)
			}
			payment.Approvers = approvers
		}
		tx := s.db.Begin()
		//check receiver and project assign
		for _, project := range projects {
			receiverIsMember := false
			for _, member := range project.Members {
				if member.MemberId == payment.ReceiverId {
					receiverIsMember = true
					break
				}
			}
			//if not is member, insert to member
			if !receiverIsMember {
				//get receiver user info
				userInfo, err := s.GetUserInfo(payment.ReceiverId)
				if err != nil {
					userInfo = storage.User{
						Id:          payment.ReceiverId,
						UserName:    payment.ReceiverName,
						DisplayName: payment.ReceiverDisplayName,
						Role:        utils.UserRoleNone,
					}
				}
				project.Members = append(project.Members, storage.Member{
					MemberId:    userInfo.Id,
					UserName:    userInfo.UserName,
					DisplayName: userInfo.DisplayName,
					Role:        int(userInfo.Role),
				})
				if err := tx.Save(&project).Error; err != nil {
					tx.Rollback()
					return nil, err
				}
			}
		}
		tx.Commit()
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (s *Service) GetPaymentProjects(projectIds []string) ([]storage.Project, error) {
	if len(projectIds) < 1 {
		return make([]storage.Project, 0), nil
	}
	projectIdsStr := strings.Join(projectIds, ",")
	projectIdsStr = fmt.Sprintf("(%s)", projectIdsStr)
	var projects []storage.Project
	query := fmt.Sprintf(`SELECT * FROM projects WHERE project_id IN %s AND approvers IS NOT NULL`, projectIdsStr)
	if err := s.db.Raw(query).Scan(&projects).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return make([]storage.Project, 0), nil
		}
		return nil, err
	}

	return projects, nil
}

func (s *Service) GetProjectApprovers(projectIds []string) (storage.Members, error) {
	projects, err := s.GetPaymentProjects(projectIds)
	if err != nil {
		return make(storage.Members, 0), nil
	}
	var approvers []storage.Member
	for _, project := range projects {
		for _, approver := range project.Approvers {
			exist := false
			for _, approveUser := range approvers {
				if approveUser.MemberId == approver.MemberId {
					exist = true
					break
				}
			}
			if !exist {
				approvers = append(approvers, approver)
			}
		}
	}
	return approvers, nil
}

func (s *Service) UpdatePayment(id, userId uint64, request portal.PaymentRequest) (*storage.Payment, error) {
	var payment storage.Payment
	if err := s.db.First(&payment, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(fmt.Errorf("payment not found with id %d", id), utils.ErrorNotFound)
		}
		return nil, err
	}

	if userId == 0 || request.ReceiverId == userId {
		// receiver or external update
		// allow recipient update status to sent or confirmed, or Rejected
		if payment.Status == storage.PaymentStatusSent || payment.Status == storage.PaymentStatusConfirmed || payment.Status == storage.PaymentStatusRejected {
			payment.Status = request.Status
			if request.Status == storage.PaymentStatusSent && payment.Status != request.Status {
				//update sentAt when status to sent
				payment.SentAt = time.Now()
			}
		}
		payment.TxId = request.TxId
	} else {
		// sender update
		payment.Description = request.Description
		payment.Details = request.Details
		payment.HourlyRate = request.HourlyRate
		payment.PaymentSettings = request.PaymentSettings
		payment.ShowDateOnInvoiceLine = request.ShowDateOnInvoiceLine
		payment.ShowProjectOnInvoice = request.ShowProjectOnInvoice
		if payment.ShowProjectOnInvoice {
			payment.ProjectId = request.ProjectId
			payment.ProjectName = request.ProjectName
		} else {
			payment.ProjectId = 0
			payment.ProjectName = ""
		}
		if !utils.IsEmpty(request.ReceiptImg) && request.Status == storage.PaymentStatusPaid {
			payment.ReceiptImg = request.ReceiptImg
		}
		if len(request.Details) > 0 {
			amount, err := calculateAmount(request)
			if err != nil {
				return nil, utils.NewError(err, utils.ErrorBadRequest)
			}
			payment.Amount = amount
			startDate, err := getStartDate(request)
			if err != nil {
				startDate = payment.CreatedAt
			}
			payment.StartDate = startDate
		} else {
			payment.Amount = request.Amount
			payment.StartDate = payment.CreatedAt
		}
		// use for sender update status from save as draft to sent
		if payment.Status == storage.PaymentStatusCreated {
			isReceiverIdNotEmpty := !utils.IsEmpty(request.ReceiverId)
			if request.ReceiverId != payment.ReceiverId && isReceiverIdNotEmpty {
				var receiver storage.User
				if err := s.db.Where("id = ?", request.ReceiverId).First(&receiver).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						return nil, utils.NewError(fmt.Errorf("receiver not found"), utils.ErrorBadRequest)
					}
					return nil, err
				}
				payment.ReceiverId = request.ReceiverId
				payment.ReceiverName = receiver.UserName
				payment.ReceiverDisplayName = receiver.DisplayName
				if len(payment.SenderDisplayName) == 0 {
					payment.SenderDisplayName = payment.SenderName
				}
				if len(payment.ReceiverDisplayName) == 0 {
					payment.ReceiverDisplayName = payment.ReceiverName
				}
			}

			if payment.Status != request.Status || (request.ReceiverId != payment.ReceiverId && isReceiverIdNotEmpty) {
				// update sentAt when status from draft to sent
				payment.SentAt = time.Now()
				//get project ids on payment
				projectIds := make([]string, 0)
				if len(payment.Details) > 0 {
					for _, detail := range payment.Details {
						if detail.ProjectId < 1 {
							continue
						}
						projectIdStr := fmt.Sprintf("%d", detail.ProjectId)
						if !slices.Contains(projectIds, projectIdStr) {
							projectIds = append(projectIds, projectIdStr)
						}
					}
				}
				//check approver on project
				approversList, err := s.GetProjectApprovers(projectIds)
				if err != nil {
					return nil, err
				}

				if len(approversList) > 0 {
					approvers := storage.Approvers{}
					for _, approver := range approversList {
						isApproved := false
						if approver.MemberId == payment.ReceiverId || approver.MemberId == payment.SenderId {
							isApproved = true
						}
						tempApprover := storage.Approver{
							ApproverId: approver.MemberId,
							IsApproved: isApproved,
							ShowCost:   true,
						}
						if utils.IsEmpty(approver.DisplayName) {
							tempApprover.ApproverName = approver.UserName
						} else {
							tempApprover.ApproverName = approver.DisplayName
						}
						approvers = append(approvers, tempApprover)
					}
					payment.Approvers = approvers
				} else {
					payment.Approvers = make(storage.Approvers, 0)
				}
				payment.Status = request.Status
			}
		}
		//if sender edit payment request, force off status back to received (sent)
		if payment.Status != storage.PaymentStatusCreated && payment.Status != storage.PaymentStatusPaid {
			payment.Status = storage.PaymentStatusSent
			projectIds := make([]string, 0)
			if len(payment.Details) > 0 {
				for _, detail := range payment.Details {
					if detail.ProjectId < 1 {
						continue
					}
					projectIdStr := fmt.Sprintf("%d", detail.ProjectId)
					if !slices.Contains(projectIds, projectIdStr) {
						projectIds = append(projectIds, projectIdStr)
					}
				}
			}
			//check approver on project
			approversList, err := s.GetProjectApprovers(projectIds)
			if err != nil {
				return nil, err
			}
			//Cancel any Approval status when sender edit payment (for all approvers)
			if len(approversList) > 0 {
				approvers := storage.Approvers{}
				for _, currentApprover := range approversList {
					oldDataExist := false
					var oldData storage.Approver
					for _, approver := range payment.Approvers {
						if approver.ApproverId == currentApprover.MemberId {
							oldDataExist = true
							oldData = approver
							break
						}
					}
					if oldDataExist {
						isApproved := false
						if oldData.ApproverId == payment.ReceiverId || oldData.ApproverId == payment.SenderId {
							isApproved = true
						}
						oldData.IsApproved = isApproved
						approvers = append(approvers, oldData)
					} else {
						displayName := currentApprover.UserName
						if !utils.IsEmpty(currentApprover.DisplayName) {
							displayName = currentApprover.DisplayName
						}
						approvers = append(approvers, storage.Approver{
							ApproverId:   currentApprover.MemberId,
							ApproverName: displayName,
							ShowCost:     true,
							IsApproved:   false,
						})
					}
				}
				payment.Approvers = approvers
			} else {
				payment.Approvers = make(storage.Approvers, 0)
			}
		}
		// if status is Draft, save show draft for recipient flag
		if request.Status == storage.PaymentStatusCreated {
			payment.ShowDraftRecipient = request.ShowDraftRecipient
			payment.Status = request.Status
		}
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (s *Service) GetListPayments(userId uint64, role utils.UserRole, request storage.PaymentFilter) ([]storage.Payment, int64, float64, error) {
	if request.Page != 0 {
		request.Page = request.Page - 1
	}
	var count int64
	var totalAmountUnpaid sql.NullFloat64
	// var totalReceived sql.NullFloat64
	payments := make([]storage.Payment, 0)
	builder := s.db
	buildCount := s.db.Model(&storage.Payment{})
	buildUnpaid := s.db.Model(&storage.Payment{})
	if request.RequestType == storage.PaymentTypeRequest {
		if request.HidePaid {
			builder = builder.Where("sender_id = ? AND status <> ? AND (? = 0 OR receiver_id IN (?))", userId, storage.PaymentStatusPaid, len(request.UserIds), request.UserIds)
			buildCount = buildCount.Where("sender_id = ? AND status <> ? AND (? = 0 OR receiver_id IN (?))", userId, storage.PaymentStatusPaid, len(request.UserIds), request.UserIds)
		} else {
			builder = builder.Where("sender_id = ? AND (? = 0 OR receiver_id IN (?))", userId, len(request.UserIds), request.UserIds)
			buildCount = buildCount.Where("sender_id = ? AND (? = 0 OR receiver_id IN (?))", userId, len(request.UserIds), request.UserIds)
		}
		builderUnpaid := buildUnpaid.Select("SUM(amount)").Where("status <> ?", storage.PaymentStatusPaid)
		if err := builderUnpaid.Scan(&totalAmountUnpaid).Error; err != nil {
			return nil, 0, 0, err
		}

	} else if request.RequestType == storage.PaymentTypeReminder {
		if request.HidePaid {
			builder = builder.Where("receiver_id = ? AND (? = 0 OR sender_id IN (?)) AND ((status <> ? AND status <> ?) OR (status = ? AND show_draft_recipient = ?))", userId, len(request.UserIds), request.UserIds, storage.PaymentStatusPaid, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
			buildCount = buildCount.Where("receiver_id = ? AND (? = 0 OR sender_id IN (?)) AND ((status <> ? AND status <> ?) OR (status = ? AND show_draft_recipient = ?))", userId, len(request.UserIds), request.UserIds, storage.PaymentStatusPaid, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
		} else {
			builder = builder.Where("receiver_id = ? AND (? = 0 OR sender_id IN (?)) AND (status <> ? OR (status = ? AND show_draft_recipient = ?))", userId, len(request.UserIds), request.UserIds, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
			buildCount = buildCount.Where("receiver_id = ? AND (? = 0 OR sender_id IN (?)) AND (status <> ? OR (status = ? AND show_draft_recipient = ?))", userId, len(request.UserIds), request.UserIds, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
		}
		builderUnpaid := buildUnpaid.Select("SUM(amount) as total").Where("receiver_id = ? AND (? = 0 OR sender_id IN (?)) AND ((status <> ? AND status <> ?) OR (status = ? AND show_draft_recipient = ?))", userId, len(request.UserIds), request.UserIds, storage.PaymentStatusPaid, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
		if err := builderUnpaid.Scan(&totalAmountUnpaid).Error; err != nil {
			return nil, 0, 0, err
		}
	} else if request.RequestType == storage.PaymentTypeApproval {
		var projectIds []uint64
		//Get project list whose approver is the logged in user
		projectQuery := fmt.Sprintf(`SELECT project_id FROM projects WHERE approvers @> '[{"memberId": %d}]'`, userId)
		if err := s.db.Raw(projectQuery).Scan(&projectIds).Error; err != nil {
			projectIds = make([]uint64, 0)
		}

		detailQueryParts := make([]string, 0)

		for _, projectId := range projectIds {
			part := fmt.Sprintf(`details @> '[{"projectId": %d}]'`, projectId)
			detailQueryParts = append(detailQueryParts, part)
		}

		detailPart := ""
		if len(detailQueryParts) >= 1 {
			detailPart = " OR "
			detailPart = fmt.Sprintf("%s%s", detailPart, strings.Join(detailQueryParts, " OR "))
		}
		isApprovedQuery := ""
		if !request.ShowApproved {
			isApprovedQuery = `, "isApproved": false`
		}
		countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM payments WHERE status = %d AND approvers @> '[{"approverId": %d%s}]' AND (project_id IN (SELECT project_id FROM projects WHERE approvers @> '[{"memberId": %d}]') %s)`, storage.PaymentStatusSent, userId, isApprovedQuery, userId, detailPart)
		if err := s.db.Raw(countQuery).Scan(&count).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return payments, 0, 0, nil
			}
			return nil, 0, 0, err
		}

		if request.Size == 0 {
			request.Size = int(count)
			request.Page = 1
		}

		offset := request.Page * request.Size
		query := fmt.Sprintf(`SELECT * FROM payments WHERE status = %d AND approvers @> '[{"approverId": %d%s}]' AND (project_id IN (SELECT project_id FROM projects WHERE approvers @> '[{"memberId": %d}]') %s) LIMIT %d OFFSET %d`, storage.PaymentStatusSent, userId, isApprovedQuery, userId, detailPart, request.Size, offset)
		if err := s.db.Raw(query).Scan(&payments).Order(request.Sort.Order).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return payments, 0, 0, nil
			}
			return nil, 0, 0, err
		}
		return payments, count, 0, nil
	} else {
		if role != utils.UserRoleAdmin {
			builder = builder.Where("receiver_id = ? OR sender_id = ?", userId, userId)
			buildCount = buildCount.Where("receiver_id = ? OR sender_id = ?", userId, userId)
		}
	}

	if err := buildCount.Count(&count).Error; err != nil {
		return nil, 0, 0, err
	}

	if request.Size == 0 {
		request.Size = int(count)
		request.Page = 0
	}
	offset := request.Page * request.Size
	if err := builder.Order(request.Sort.Order).Limit(request.Size).Offset(offset).Find(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, 0, 0, nil
		}
		return nil, 0, 0, err
	}

	return payments, count, totalAmountUnpaid.Float64, nil
}

func (s *Service) CheckHasReport(userId uint64) bool {
	var count int64
	err := s.db.Model(&storage.Payment{}).Where("status = ? AND receiver_id = ?", storage.PaymentStatusPaid, userId).Count(&count).Error
	if err != nil || count < 1 {
		return false
	}
	return true
}

func (s *Service) GetPaymentUserList(userId uint64) ([]storage.User, error) {
	var result []storage.User
	query := fmt.Sprintf(`SELECT * FROM public.users WHERE id IN (SELECT receiver_id FROM payments WHERE sender_id = %d) OR id IN (SELECT sender_id FROM payments WHERE receiver_id = %d)`, userId, userId)
	if err := s.db.Raw(query).Scan(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return make([]storage.User, 0), nil
		}
		return nil, err
	}
	return result, nil
}

func (s *Service) GetProjectRelatedMembers(userId uint64) ([]storage.User, error) {
	var projects []storage.Project
	query := fmt.Sprintf(`SELECT * FROM public.projects WHERE creator_id = %d`, userId)
	if err := s.db.Raw(query).Scan(&projects).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return make([]storage.User, 0), nil
		}
		return nil, err
	}
	result := make([]storage.User, 0)
	for _, project := range projects {
		members := project.Members
		members = append(members, project.Approvers...)
		for _, member := range members {
			exist := false
			for _, user := range result {
				if user.Id == member.MemberId {
					exist = true
					break
				}
			}
			if !exist {
				user, err := s.GetUserInfo(member.MemberId)
				if err == nil {
					result = append(result, user)
				}
			}
		}
	}
	return result, nil
}

func (s *Service) GetPaymentsForReport(userId uint64, request portal.ReportFilter) ([]storage.Payment, error) {
	payments := make([]storage.Payment, 0)
	var memberQuery = ""
	var projectQuery = ""
	if !utils.IsEmpty(request.MemberIds) {
		memberQuery = fmt.Sprintf(`AND sender_id IN (%s)`, request.MemberIds)
	}
	if !utils.IsEmpty(request.ProjectIds) {
		var orQuery = ""
		var projectIdArr = strings.Split(request.ProjectIds, ",")
		for index, projectId := range projectIdArr {
			if index == 0 {
				orQuery = fmt.Sprintf(`details @> '[{"projectId": %s}]'`, projectId)
			} else {
				orQuery = fmt.Sprint(orQuery, fmt.Sprintf(` OR details @> '[{"projectId": %s}]'`, projectId))
			}
		}
		projectQuery = fmt.Sprintf(`AND (%s)`, orQuery)
	}

	query := fmt.Sprintf(`SELECT * FROM payments WHERE status = %d AND (paid_at AT TIME ZONE 'UTC') < '%s' AND (paid_at AT TIME ZONE 'UTC') > '%s' AND receiver_id = %d %s %s ORDER BY paid_at DESC`,
		storage.PaymentStatusPaid, utils.TimeToStringWithoutTimeZone(request.EndDate), utils.TimeToStringWithoutTimeZone(request.StartDate), userId, memberQuery, projectQuery)
	if err := s.db.Raw(query).Scan(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, nil
		}
		return nil, err
	}
	return payments, nil
}

// get all payments exculed draft status
func (s *Service) GetAllPayments(request storage.AdminReportFilter) ([]storage.Payment, error) {
	payments := make([]storage.Payment, 0)
	query := fmt.Sprintf(`SELECT * FROM payments WHERE status <> %d AND status <> %d AND (sent_at AT TIME ZONE 'UTC') < '%s' AND (sent_at AT TIME ZONE 'UTC') > '%s' ORDER BY sent_at DESC`,
		storage.PaymentStatusCreated, storage.PaymentStatusRejected, utils.TimeToStringWithoutTimeZone(request.EndDate), utils.TimeToStringWithoutTimeZone(request.StartDate))
	if err := s.db.Raw(query).Scan(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, nil
		}
		return nil, err
	}
	return payments, nil
}

func (s *Service) GetForInvoiceReport(userId uint64, request portal.ReportFilter) ([]storage.Payment, error) {
	payments := make([]storage.Payment, 0)
	var memberQuery = ""
	var projectQuery = ""
	if !utils.IsEmpty(request.MemberIds) {
		memberQuery = fmt.Sprintf(`AND sender_id IN (%s)`, request.MemberIds)
	}
	if !utils.IsEmpty(request.ProjectIds) {
		var orQuery = ""
		var projectIdArr = strings.Split(request.ProjectIds, ",")
		for index, projectId := range projectIdArr {
			if index == 0 {
				orQuery = fmt.Sprintf(`details @> '[{"projectId": %s}]'`, projectId)
			} else {
				orQuery = fmt.Sprint(orQuery, fmt.Sprintf(` OR details @> '[{"projectId": %s}]'`, projectId))
			}
		}
		projectQuery = fmt.Sprintf(`AND (%s)`, orQuery)
	}

	query := fmt.Sprintf(`SELECT * FROM payments WHERE status = %d AND paid_at < '%s' AND paid_at > '%s' AND details @> '[{"price": 0}]' AND receiver_id = %d %s %s ORDER BY paid_at DESC`,
		storage.PaymentStatusPaid, utils.TimeToStringWithoutTimeZone(request.EndDate), utils.TimeToStringWithoutTimeZone(request.StartDate), userId, memberQuery, projectQuery)
	if err := s.db.Raw(query).Scan(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, nil
		}
		return nil, err
	}
	return payments, nil
}

func (s *Service) GetApprovalsCount(userId uint64, showApproved bool) (int64, error) {
	var projectIds []uint64
	//Get project list whose approver is the logged in user
	projectQuery := fmt.Sprintf(`SELECT project_id FROM projects WHERE approvers @> '[{"memberId": %d}]'`, userId)
	if err := s.db.Raw(projectQuery).Scan(&projectIds).Error; err != nil {
		projectIds = make([]uint64, 0)
	}
	detailQueryParts := make([]string, 0)
	for _, projectId := range projectIds {
		part := fmt.Sprintf(`details @> '[{"projectId": %d}]'`, projectId)
		detailQueryParts = append(detailQueryParts, part)
	}

	detailPart := ""
	if len(detailQueryParts) >= 1 {
		detailPart = " OR "
		detailPart = fmt.Sprintf("%s%s", detailPart, strings.Join(detailQueryParts, " OR "))
	}
	isApprovedQuery := ""
	if !showApproved {
		isApprovedQuery = `, "isApproved": false`
	}
	var count int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM payments WHERE status = %d AND approvers @> '[{"approverId": %d%s}]' AND (project_id IN (SELECT project_id FROM projects WHERE approvers @> '[{"memberId": %d}]') %s)`, storage.PaymentStatusSent, userId, isApprovedQuery, userId, detailPart)
	if err := s.db.Raw(countQuery).Scan(&count).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (s *Service) GetUnpaidCount(userId uint64) (int64, error) {
	var count int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM payments WHERE status <> %d AND status <> %d AND status <> %d AND receiver_id = %d`, storage.PaymentStatusPaid, storage.PaymentStatusRejected, storage.PaymentStatusCreated, userId)
	if err := s.db.Raw(countQuery).Scan(&count).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (s *Service) BulkPaidBTC(userId uint64, txId string, bulkPays []portal.BulkPaymentBTC) error {
	paymentIds := make([]int, 0)
	bulkMap := make(map[int]portal.BulkPaymentBTC)

	for _, pay := range bulkPays {
		paymentIds = append(paymentIds, pay.ID)
		bulkMap[pay.ID] = pay
	}

	payments := make([]*storage.Payment, 0)
	if err := s.db.Where("id IN ?", paymentIds).Find(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	// validate payment
	for _, paym := range payments {
		if paym.Status != storage.PaymentStatusConfirmed && paym.Status != storage.PaymentStatusSent {
			return fmt.Errorf("%s", "all payments need to be ready for payment")
		}

		if paym.ReceiverId != userId {
			return fmt.Errorf("%s", "all payments must be yours")
		}
		if len(paym.PaymentSettings) <= 0 {
			return fmt.Errorf("%s", "Get payment method list failed")
		}

		hasBTCMethod := false
		btcAddress := ""
		for _, paySetting := range paym.PaymentSettings {
			if paySetting.Type == utils.PaymentTypeBTC {
				hasBTCMethod = true
				btcAddress = paySetting.Address
				break
			}
		}
		if !hasBTCMethod {
			return fmt.Errorf("%s", "Payment is not set to pay for BTC")
		}

		paym.PaymentMethod = utils.PaymentTypeBTC
		paym.PaymentAddress = btcAddress
		paym.TxId = txId
		paym.PaidAt = time.Now()
		paym.Status = storage.PaymentStatusPaid
	}

	//Update data
	for _, pay := range payments {
		id := int(pay.Id)
		pay.ConvertRate = bulkMap[id].Rate
		pay.ConvertTime = time.Unix(bulkMap[id].ConvertTime, 0)
		pay.TxId = txId
	}

	if err := s.db.Save(&payments).Error; err != nil {
		fmt.Println("ERROR: ", err)
		return &utils.InternalError
	}
	return nil
}

func calculateAmount(request portal.PaymentRequest) (float64, error) {
	var amount float64
	for i, detail := range request.Details {
		if detail.Quantity > 0 {
			var price = request.HourlyRate
			if detail.Price > 0 {
				price = detail.Price
			}
			cost := detail.Quantity * price
			if cost != detail.Cost {
				return 0, fmt.Errorf("payment detail amount is incorrect at line %d", i+1)
			}
			if detail.Cost <= 0 {
				return 0, fmt.Errorf("payment detail cost must be greater than 0 at line %d", i+1)
			}
		}
		amount += detail.Cost
	}
	return amount, nil
}

func getStartDate(request portal.PaymentRequest) (time.Time, error) {
	var start_date = time.Now().AddDate(1000, 0, 0)
	hasStartDate := false
	for _, detail := range request.Details {
		fullFormatDate := GetFullFormatDate(detail.Date)
		parse_date, err := time.Parse("2006/01/02", fullFormatDate)
		if err == nil && parse_date.Before(start_date) {
			start_date = parse_date
			hasStartDate = true
		}
	}
	if !hasStartDate {
		return time.Now(), fmt.Errorf("%s", "Don't have start date on details")
	}
	return start_date, nil
}

func GetFullFormatDate(inputDate string) string {
	if utils.IsEmpty(inputDate) {
		return inputDate
	}
	dateArr := strings.Split(inputDate, "/")
	if len(dateArr) < 3 {
		return inputDate
	}
	year := dateArr[0]
	month := dateArr[1]
	day := dateArr[2]
	if strings.HasPrefix(day, "0") {
		return inputDate
	}
	dayNumber, intErr := strconv.ParseInt(day, 0, 32)
	if intErr != nil || dayNumber == 0 {
		return inputDate
	}
	if dayNumber >= 10 {
		return inputDate
	}
	dayDisp := fmt.Sprintf("0%d", dayNumber)
	return fmt.Sprintf("%s/%s/%s", year, month, dayDisp)
}

// Sync Payment data when user Display name was changed
func (s *Service) SyncPaymentUser(db *gorm.DB, uID int, displayName, userName string) error {
	//update displayname for every payment request current user is sender or receiver
	updateSenderBuilder := db.Model(&storage.Payment{}).
		Where("sender_id = ? AND status NOT IN (?,?) AND created_at >= date_trunc('month', now()) - interval '3 month'", uID, storage.PaymentStatusPaid, storage.PaymentStatusRejected)

	updatereceiverBuilder := db.Model(&storage.Payment{}).Where("receiver_id = ? AND status NOT IN (?,?) AND created_at >= date_trunc('month', now()) - interval '3 month'", uID, storage.PaymentStatusPaid, storage.PaymentStatusRejected)

	if !utils.IsEmpty(displayName) {
		if err := updateSenderBuilder.UpdateColumn("sender_display_name", displayName).Error; err != nil {
			return err
		}
		if err := updatereceiverBuilder.UpdateColumn("receiver_display_name", displayName).Error; err != nil {
			return err
		}
	}

	if !utils.IsEmpty(userName) {
		if err := updateSenderBuilder.UpdateColumn("sender_name", userName).Error; err != nil {
			return err
		}
		if err := updatereceiverBuilder.UpdateColumn("receiver_name", userName).Error; err != nil {
			return err
		}
	}
	return nil
}
