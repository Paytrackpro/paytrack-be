package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func ValidateFilter(validMapFields map[string]bool, sortStr string) error {
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
		if _, ok := validMapFields[orderInfo[0]]; !ok {
			var validFields []string
			for f, _ := range validMapFields {
				validFields = append(validFields, f)
			}
			return fmt.Errorf("invalid field name: '%s', valid field name is: '%s'", orderInfo[0], strings.Join(validFields, ","))
		}
	}
	return nil
}
