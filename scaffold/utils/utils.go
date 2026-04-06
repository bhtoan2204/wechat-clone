package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var initialisms = map[string]string{
	"id":  "ID",
	"ids": "IDs",
	"url": "URL",
}

func ZeroCheck(typ string, field string) string {
	switch typ {
	case "string":
		return fmt.Sprintf("r.%s == \"\"", field)
	case "*string":
		return fmt.Sprintf("r.%s == nil || *r.%s == \"\"", field, field)
	case "int", "int64":
		return fmt.Sprintf("r.%s == 0", field)
	case "bool":
		return fmt.Sprintf("r.%s == false", field)
	default:
		if strings.HasPrefix(typ, "[]") || strings.HasPrefix(typ, "map[") {
			return fmt.Sprintf("len(r.%s) == 0", field)
		}
		return ""
	}
}

func Snake(name string) string {
	name = strings.TrimSuffix(name, "Request")
	name = strings.TrimSuffix(name, "Response")
	name = strings.TrimSuffix(name, "Handler")
	var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := matchFirstCap.ReplaceAllString(name, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func Pascal(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if initialism, ok := initialisms[strings.ToLower(p)]; ok {
			parts[i] = initialism
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func GoType(t string) string {
	switch t {
	case "string":
		return "string"
	case "int":
		return "int"
	case "int64":
		return "int64"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}
