package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/datawire/apro/lib/filterapi"
)

func NewErrorResponse(ctx context.Context, httpStatus int, err error, extra map[string]interface{}) *filterapi.HTTPResponse {
	body := map[string]interface{}{
		"status_code": httpStatus,
		"message":     err.Error(),
	}
	if httpStatus/100 == 5 {
		body["request_id"] = GetRequestID(ctx)
	}
	for k, v := range extra {
		if _, set := body[k]; !set {
			body[k] = v
		}
	}
	bodyBytes, _ := json.Marshal(body)
	if httpStatus/100 == 5 {
		GetLogger(ctx).Errorf("HTTP %v %+v", httpStatus, err)
	} else {
		GetLogger(ctx).Infof("HTTP %v %+v", httpStatus, err)
	}
	return &filterapi.HTTPResponse{
		StatusCode: httpStatus,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
		Body: string(bodyBytes),
	}
}
