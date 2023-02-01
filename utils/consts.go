package utils

const (
	SortByCreated  = 1
	SortByLastSeen = 2

	SortASC  = 1
	SortDESC = 2
)

type UserRole int

const (
	UserRoleNone UserRole = iota
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
