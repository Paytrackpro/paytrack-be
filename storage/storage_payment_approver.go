package storage

import (
	"gorm.io/gorm"
)

type PaymentApprover struct {
	Id         uint64 `gorm:"primarykey" json:"id"`
	SenderId   uint64 `json:"senderId"`
	ReceiverId uint64 `json:"receiverId"`
	ApproverId uint64 `json:"approverId"`
}

type PaymentApproverFilter struct {
	Sort
}

type PaymentApprovalStatus struct {
	Id                uint64 `gorm:"primarykey" json:"id"`
	PaymentId         uint64 `json:"paymentId"`
	PaymentApproverId uint64 `json:"paymentApproverId"`
	Status            bool   `json:"status"`
}

func (f *PaymentApproverFilter) selectFields(db *gorm.DB) *gorm.DB {
	return db.Select("payment_approvers.*").
		Joins("left join payments p on p.sender_id = payment_approvers.sender_id and p.receiver_id = payment_approvers.receiver_id")
}

func (f *PaymentApproverFilter) BindCount(db *gorm.DB) *gorm.DB {
	return db
}

func (f *PaymentApproverFilter) BindQuery(db *gorm.DB) *gorm.DB {
	db = f.selectFields(db)
	db = f.Sort.BindQuery(db)
	return f.BindCount(db)
}

func (f *PaymentApproverFilter) BindFirst(db *gorm.DB) *gorm.DB {
	return db
}

func (f *PaymentApproverFilter) Sortable() map[string]bool {
	return map[string]bool{
		"createdAt": true,
		"paidAt":    true,
		"status":    true,
		"amount":    true,
	}
}
