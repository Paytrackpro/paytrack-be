package storage

import (
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
	"strings"
)

type Sort struct {
	sortableFields []string
	Order          string
	Page           int
	Size           int
}

const defaultOffset = 20

func (s *Sort) RequestedSort() string {
	return s.Order
}

func (s *Sort) BindQuery(db *gorm.DB) *gorm.DB {
	if s.Page < 0 {
		s.Page = 1
	}
	if s.Size <= 0 {
		s.Size = defaultOffset
	}
	offset := (s.Page - 1) * s.Size
	db = db.Limit(s.Size).Offset(offset)
	var order = strings.Trim(s.Order, " ")
	if len(order) > 0 {
		var orders = strings.Split(order, ",")
		for i, order := range orders {
			orders[i] = utils.ToSnakeCase(order)
		}
		return db.Order(strings.Join(orders, ","))
	}
	return db
}
