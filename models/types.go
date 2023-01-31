package models

type MSort struct {
	SortType int
	Sort     int
	Limit    int
	Offset   int
}
type UserFilter struct {
	MSort
	KeySearch string
}
