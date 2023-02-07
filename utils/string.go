package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// ToSnakeCase replace the camel string into snake case, used for getting table field from struct field
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ValidateSortField checks the fields is valid or not to be sorted on the database
func ValidateSortField(validMapFields map[string]bool, sortStr string) error {
	sortStr = strings.TrimSpace(sortStr)
	if len(sortStr) == 0 {
		return nil
	}
	if validMapFields == nil && len(sortStr) != 0 {
		return fmt.Errorf("sort is not supported")
	}
	fields := strings.Split(sortStr, ",")
	for _, field := range fields {
		orderInfo := strings.Split(field, " ")
		if len(orderInfo) > 2 ||
			(len(orderInfo) == 2 && orderInfo[1] != "desc" && orderInfo[1] != "asc") {

			return fmt.Errorf("invalid format: '%s', expected formart is 'fied_name desc/asc'", field)
		}
		if validMapFields[orderInfo[0]] {
			var validFields []string
			for f := range validMapFields {
				validFields = append(validFields, f)
			}
			return fmt.Errorf("invalid field name: '%s', valid field name is: '%s'", orderInfo[0], strings.Join(validFields, ","))
		}
	}
	return nil
}
