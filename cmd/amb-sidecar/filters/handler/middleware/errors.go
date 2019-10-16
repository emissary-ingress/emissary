package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"text/template"

	"github.com/datawire/apro/lib/filterapi"
)

func errorResponse(ctx context.Context, httpStatus int, err error, extra map[string]interface{}) []byte {
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
	return bodyBytes
}

func TemplatedErrorResponse(ctx context.Context, httpStatus int, err error, errorTemplate template.Template, contentType string) *filterapi.HTTPResponse {
	rawError := map[string]interface{}{
		"httpStatus": httpStatus,
		"error":      err,
		"requestId":  GetRequestID(ctx),
	}

	bodyBuffer := &bytes.Buffer{}
	templateErr := errorTemplate.Execute(bodyBuffer, rawError)
	if templateErr != nil {
		GetLogger(ctx).Errorf("Failed to render response from template %s", templateErr)
	}

	GetLogger(ctx).Infof("HTTP %v %+v", httpStatus, err)
	return &filterapi.HTTPResponse{
		StatusCode: httpStatus,
		Header: http.Header{
			"Content-Type": {contentType},
		},
		Body: bodyBuffer.String(),
	}
}

func NewErrorResponse(ctx context.Context, httpStatus int, err error, extra map[string]interface{}) *filterapi.HTTPResponse {
	bodyBytes := errorResponse(ctx, httpStatus, err, extra)

	return &filterapi.HTTPResponse{
		StatusCode: httpStatus,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
		Body: string(bodyBytes),
	}
}

func ServeErrorResponse(w http.ResponseWriter, ctx context.Context, httpStatus int, err error, extra map[string]interface{}) {
	bodyBytes := errorResponse(ctx, httpStatus, err, extra)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	w.Write(bodyBytes)
}
