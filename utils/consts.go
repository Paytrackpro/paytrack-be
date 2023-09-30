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

type ProductStatus uint32

const (
	Hidden ProductStatus = iota
	Active
	Deleted
)
