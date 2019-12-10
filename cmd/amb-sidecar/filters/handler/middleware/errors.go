package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/lib/filterapi"
)

var defaultTemplate = func() crd.ErrorResponse {
	var ret crd.ErrorResponse
	if err := ret.Validate(""); err != nil {
		panic(err)
	}
	return ret
}()

type errorData map[string]interface{}

func (ed errorData) MarshalJSON() ([]byte, error) {
	copy := make(map[string]interface{})
	for k, v := range ed {
		switch k {
		case "request_id":
			httpStatus, httpStatusOK := ed["status_code"].(int)
			if httpStatusOK && httpStatus/100 == 5 {
				copy[k] = v
			}
		case "httpStatus", "requestId", "error":
			// skip, these are for compatibility with older custom templates
		default:
			copy[k] = v
		}
	}
	return json.Marshal(copy)
}

func errorResponse(tmpl *crd.ErrorResponse, ctx context.Context, httpStatus int, err error, extra map[string]interface{}) (http.Header, []byte) {
	if tmpl == nil {
		tmpl = &defaultTemplate
	}

	bodyData := errorData{
		"status_code": httpStatus,
		"httpStatus":  httpStatus, // for user-template backward-compatibility
		"message":     err.Error(),
		"error":       err, // for user-template backward-compatibility
		"request_id":  GetRequestID(ctx),
		"requestId":   GetRequestID(ctx), // for user-template backward-compatibility
	}
	for k, v := range extra {
		if _, set := bodyData[k]; !set {
			bodyData[k] = v
		}
	}

	header := make(http.Header)
	for _, field := range tmpl.Headers {
		var value strings.Builder
		if err := field.Template.Execute(&value, bodyData); err != nil {
			return errorResponse(nil, ctx, 500,
				errors.Wrapf(err, "errorResponse: generating header %q", field.Name), nil)
		}
		header.Set(field.Name, value.String())
	}

	var body bytes.Buffer
	if err := tmpl.BodyTemplate.Execute(&body, bodyData); err != nil {
		return errorResponse(nil, ctx, 500,
			errors.Wrap(err, "errorResponse: generating body"), nil)
	}

	if httpStatus/100 == 5 {
		dlog.GetLogger(ctx).Errorf("HTTP %v %+v", httpStatus, err)
	} else {
		dlog.GetLogger(ctx).Infof("HTTP %v %+v", httpStatus, err)
	}

	return header, body.Bytes()
}

func NewTemplatedErrorResponse(tmpl *crd.ErrorResponse, ctx context.Context, httpStatus int, err error, extra map[string]interface{}) *filterapi.HTTPResponse {
	header, bodyBytes := errorResponse(tmpl, ctx, httpStatus, err, extra)

	return &filterapi.HTTPResponse{
		StatusCode: httpStatus,
		Header:     header,
		Body:       string(bodyBytes),
	}
}

func NewErrorResponse(ctx context.Context, httpStatus int, err error, extra map[string]interface{}) *filterapi.HTTPResponse {
	return NewTemplatedErrorResponse(nil, ctx, httpStatus, err, extra)
}

func ServeErrorResponse(w http.ResponseWriter, ctx context.Context, httpStatus int, err error, extra map[string]interface{}) {
	header, bodyBytes := errorResponse(nil, ctx, httpStatus, err, extra)
	for k, vs := range header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(httpStatus)
	w.Write(bodyBytes)
}
