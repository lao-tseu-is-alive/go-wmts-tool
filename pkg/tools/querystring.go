package tools

import "fmt"

// BuildQueryString builds a URL query string from a map of parameters.
func BuildQueryString(params map[string]string) string {
	queryString := ""
	first := true
	for k, v := range params {
		if !first {
			queryString += "&"
		}
		queryString += fmt.Sprintf("%s=%s", k, v)
		first = false
	}
	return queryString
}
