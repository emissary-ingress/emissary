package main

import (
	"fmt"
	"net/http"
	"strings"
)

func SetHeaders(r *http.Request, headers map[string]string) {
	for k, v := range headers {
		r.Header.Set(k, v)
	}
}

func FormatCookieHeaderFromCookieMap(cookies map[string]string) (result string) {
	for k, v := range cookies {
		result += fmt.Sprintf("%s=%s;", k, v)
	}

	result = strings.TrimSuffix(result, ";")

	return
}

func ExtractCookies(response *http.Response, names []string) (result map[string]string, err error) {
	result = make(map[string]string)

	for _, cookie := range response.Cookies() {
		if pos, contained := contains(cookie.Name, names); contained {
			result[cookie.Name] = cookie.Value
			remove(pos, names)
		}
	}

	//if len(names) != 0 {
	//	err = fmt.Errorf("not all cookies found: %v\n", names)
	//}

	return
}

func contains(value string, items []string) (int, bool) {
	for idx, item := range items {
		if item == value {
			return idx, true
		}
	}

	return -1, false
}

func remove(pos int, src []string) []string {
	src[len(src)-1], src[pos] = src[pos], src[len(src)-1]
	return src[:len(src)-1]
}
