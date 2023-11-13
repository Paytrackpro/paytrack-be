package service

import (
	"database/sql"
	"fmt"
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
	offset := page * pageSize

	build :=
		s.db.Model(&storage.Payment{}).Order(order).
			Where("payments.payment_method = ? AND payments.status = ? AND payments.receiver_id = ?", utils.PaymentTypeBTC, storage.PaymentStatusConfirmed, userId).
			Scan(&payments)

	buildCount := s.db.Model(&storage.Payment{}).Where("payment_method = ? AND status = ? AND receiver_id = ?", utils.PaymentTypeBTC, storage.PaymentStatusConfirmed, userId)
	if err := buildCount.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	build = build.Limit(pageSize).Offset(offset)
	if err := build.Find(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, 0, nil
		}
		return nil, 0, err
	}

	return payments, count, nil
}

func (s *Service) CountBulkPaymentBTC(userId uint64) (int64, error) {
	var count int64
	buildCount := s.db.Model(&storage.Payment{}).Where("payment_method = ? AND status = ? AND receiver_id = ?", utils.PaymentTypeBTC, storage.PaymentStatusConfirmed, userId)
	if err := buildCount.Count(&count).Error; err != nil {
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

func (s *Service) CreatePayment(userId uint64, userName string, displayName string, showDateOnInvoiceLine bool, showDraftForRecipient bool, request portal.PaymentRequest) (*storage.Payment, error) {
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
		ShowDateOnInvoiceLine: showDateOnInvoiceLine,
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
	} else {
		payment.Amount = request.Amount
	}

	if payment.Status == storage.PaymentStatusSent {
		//if status is sent, set sentAt is now
		payment.SentAt = time.Now()
		approverSettings, err := s.GetApproverForPayment(userId, payment.ReceiverId)
		if err != nil {
			return nil, err
		}

		if len(approverSettings) > 0 {
			approvers := storage.Approvers{}
			for _, approver := range approverSettings {
				approvers = append(approvers, storage.Approver{
					ApproverId:   approver.ApproverId,
					ApproverName: approver.ApproverName,
					IsApproved:   false,
					ShowCost:     approver.ShowCost,
				})
			}
			payment.Approvers = approvers
		}
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
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
		if !utils.IsEmpty(request.ReceiptImg) && request.Status == storage.PaymentStatusPaid {
			payment.ReceiptImg = request.ReceiptImg
		}
		if len(request.Details) > 0 {
			amount, err := calculateAmount(request)
			if err != nil {
				return nil, utils.NewError(err, utils.ErrorBadRequest)
			}
			payment.Amount = amount
		} else {
			payment.Amount = request.Amount
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
				approverSettings, err := s.GetApproverForPayment(userId, payment.ReceiverId)
				if err != nil {
					return nil, err
				}

				if len(approverSettings) > 0 {
					approvers := storage.Approvers{}
					for _, approver := range approverSettings {
						approvers = append(approvers, storage.Approver{
							ApproverId:   approver.ApproverId,
							ApproverName: approver.ApproverName,
							IsApproved:   false,
							ShowCost:     approver.ShowCost,
						})
					}
					payment.Approvers = approvers
				}
				payment.Status = request.Status
			}
		}
		//if sender edit payment request, force off status back to received (sent)
		if payment.Status != storage.PaymentStatusCreated && payment.Status != storage.PaymentStatusPaid {
			payment.Status = storage.PaymentStatusSent
			approverSettings, err := s.GetApproverForPayment(userId, payment.ReceiverId)
			if err != nil {
				return nil, err
			}
			//Cancel any Approval status when sender edit payment (for all approvers)
			if len(approverSettings) > 0 {
				for i, _ := range payment.Approvers {
					payment.Approvers[i].IsApproved = false
				}
			}
		}
		// if status is Draft, save show draft for recipient flag
		if request.Status == storage.PaymentStatusCreated {
			payment.ShowDraftRecipient = request.ShowDraftRecipient
		}
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (s *Service) GetListPayments(userId uint64, role utils.UserRole, request storage.PaymentFilter) ([]storage.Payment, int64, error) {
	if request.Page != 0 {
		request.Page = request.Page - 1
	}
	var count int64
	payments := make([]storage.Payment, 0)
	offset := request.Page * request.Size
	builder := s.db
	buildCount := s.db.Model(&storage.Payment{})
	if request.RequestType == storage.PaymentTypeRequest {
		if request.HidePaid {
			builder = builder.Where("sender_id = ? AND status <> ?", userId, storage.PaymentStatusPaid)
			buildCount = buildCount.Where("sender_id = ? AND status <> ?", userId, storage.PaymentStatusPaid)
		} else {
			builder = builder.Where("sender_id = ?", userId)
			buildCount = buildCount.Where("sender_id = ?", userId)
		}
	} else if request.RequestType == storage.PaymentTypeReminder {
		if request.HidePaid {
			builder = builder.Where("receiver_id = ? AND ((status <> ? AND status <> ?) OR (status = ? AND show_draft_recipient = ?))", userId, storage.PaymentStatusPaid, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
			buildCount = buildCount.Where("receiver_id = ? AND ((status <> ? AND status <> ?) OR (status = ? AND show_draft_recipient = ?))", userId, storage.PaymentStatusPaid, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
		} else {
			builder = builder.Where("receiver_id = ? AND (status <> ? OR (status = ? AND show_draft_recipient = ?))", userId, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
			buildCount = buildCount.Where("receiver_id = ? AND (status <> ? OR (status = ? AND show_draft_recipient = ?))", userId, storage.PaymentStatusCreated, storage.PaymentStatusCreated, true)
		}
	} else if request.RequestType == storage.PaymentTypeApproval {
		query := fmt.Sprintf(`SELECT * FROM payments WHERE status = %d AND approvers @> '[{"approverId": %d, "isApproved": false}]' LIMIT %d OFFSET %d`, storage.PaymentStatusSent, userId, request.Size, offset)
		countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM payments WHERE status = %d AND approvers @> '[{"approverId": %d, "isApproved": false}]'`, storage.PaymentStatusSent, userId)
		if err := s.db.Raw(query).Scan(&payments).Order(request.Sort.Order).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return payments, 0, nil
			}
			return nil, 0, err
		}

		if err := s.db.Raw(countQuery).Scan(&count).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return payments, 0, nil
			}
			return nil, 0, err
		}
		return payments, count, nil
	} else {
		if role != utils.UserRoleAdmin {
			builder = builder.Where("receiver_id = ? OR sender_id = ?", userId, userId)
			buildCount = buildCount.Where("receiver_id = ? OR sender_id = ?", userId, userId)
		}
	}

	if err := buildCount.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	if err := builder.Order(request.Sort.Order).Limit(request.Size).Offset(offset).Find(&payments).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return payments, 0, nil
		}
		return nil, 0, err
	}

	return payments, count, nil
}

func (s *Service) GetApprovalsCount(userId uint64) (int64, error) {
	var count int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM payments WHERE status = %d AND approvers @> '[{"approverId": %d, "isApproved": false}]'`, storage.PaymentStatusSent, userId)
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
		if paym.Status != storage.PaymentStatusConfirmed {
			return fmt.Errorf("all payments need to be ready for payment")
		}

		if paym.ReceiverId != userId {
			return fmt.Errorf("all payments must be yours")
		}

		if paym.PaymentMethod != utils.PaymentTypeBTC {
			return fmt.Errorf("all payments needs the payment method to be BTC")
		}
		paym.TxId = txId
		paym.PaidAt = time.Now()
		paym.Status = storage.PaymentStatusPaid
	}

	//Update data
	for _, pay := range payments {
		id := int(pay.Id)
		pay.ConvertRate = bulkMap[id].Rate
		pay.PaymentMethod = bulkMap[id].PaymentMethod
		pay.ConvertTime = time.Unix(bulkMap[id].ConvertTime, 0)
		pay.PaymentAddress = bulkMap[id].PaymentAddress
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
