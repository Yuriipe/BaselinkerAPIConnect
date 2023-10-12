package payload

import (
	"net/url"
	"strings"
)

func Creator(method string, parameters string) string {
	payload := url.Values{}
	p := strings.Split(parameters, "")
	payload.Add("method", method)
	if len(p) > 0 {
		payload.Add("parameters", parameters)
	}
	return payload.Encode()
}
