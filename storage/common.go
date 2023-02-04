package storage

import "gorm.io/gorm"

type Sort struct {
	sortableFields []string
	Order          string
	Page           int
	Size           int
}

const defaultOffset = 20

func (s *Sort) BindQuery(db *gorm.DB) *gorm.DB {
	if s.Page < 0 {
		s.Page = 1
	}
	if s.Size <= 0 {
		s.Size = defaultOffset
	}
	offset := (s.Page - 1) * s.Size
	db = db.Limit(s.Size).Offset(offset)
	return db
}

func (s *Sort) With(f Filter) *Sort {
	s.sortableFields = f.Sortable()
	return s
}
