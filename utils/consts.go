package utils

const (
	SortByCreated  = 1
	SortByLastSeen = 2

	SortASC  = 1
	SortDESC = 2
)

const AuthClaimsCtxKey = "authClaimsCtxKey"

type UserRole int

const (
	UserRoleNone UserRole = iota
	UserRoleManager
	UserRoleAdmin
)

// User status
type UserStatus int

const (
	StatusAll UserStatus = iota
	StatusWaitApproved
	Hired
	Quit
)

type PaymentType int

const (
	PaymentTypeNotSet PaymentType = iota
	PaymentTypeBTC
	PaymentTypeLTC
	PaymentTypeDCR
)
