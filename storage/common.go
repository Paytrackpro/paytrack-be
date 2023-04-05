package storage

import (
	"strings"

	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
)

type Sort struct {
	Order string `schema:"order"`
	Page  int    `schema:"page"`
	Size  int    `schema:"size"`
}

const defaultOffset = 20

func (s *Sort) RequestedSort() string {
	return s.Order
}

func (s *Sort) BindQuery(db *gorm.DB) *gorm.DB {
	if s.Page <= 0 {
		s.Page = 1
	}
	if s.Size <= 0 {
		s.Size = defaultOffset
	}
	offset := (s.Page - 1) * s.Size
	db = db.Limit(s.Size).Offset(offset)
	var order = strings.TrimSpace(s.Order)
	if len(order) > 0 {
		var orders = strings.Split(order, ",")
		for i, order := range orders {
			orders[i] = utils.ToSnakeCase(order)
		}
		return db.Order(strings.Join(orders, ","))
	}
	return db
}
