package parser

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type HTTPRequest struct {
	ID         bson.ObjectID          `bson:"_id,omitempty" json:"id"`
	Method     string                 `bson:"method" json:"method"`
	Path       string                 `bson:"path" json:"path"`
	GetParams  map[string]interface{} `bson:"get_params" json:"get_params"`
	Headers    map[string]string      `bson:"headers" json:"headers"`
	Cookies    map[string]interface{} `bson:"cookies" json:"cookies"`
	PostParams map[string]string      `bson:"post_params" json:"post_params"`
	Timestamp  time.Time              `bson:"timestamp" json:"timestamp"`
}

type HTTPResponse struct {
	ID        bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	RequestID bson.ObjectID     `bson:"request_id,omitempty" json:"request_id"`
	Code      int               `bson:"code" json:"code"`
	Message   string            `bson:"message" json:"message"`
	Headers   map[string]string `bson:"headers" json:"headers"`
	Body      string            `bson:"body" json:"body"`
	Timestamp time.Time         `bson:"timestamp" json:"timestamp"`
}

func ParseRequest(req *http.Request, bodyBytes []byte) (*HTTPRequest, error) {
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	getParams := parseQueryParams(req.URL.Query())

	headers := make(map[string]string)
	for k, v := range req.Header {
		headers[k] = strings.Join(v, ", ")
	}

	cookies := make(map[string]interface{})
	for _, c := range req.Cookies() {
		cookies[c.Name] = tryParseInt(c.Value)
	}

	postParams := make(map[string]string)
	if req.Method == http.MethodPost {
		if strings.Contains(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			if err := req.ParseForm(); err == nil {
				for k, v := range req.PostForm {
					postParams[k] = v[0]
				}
			}
		} else if len(bodyBytes) > 0 {
			for _, line := range strings.Split(string(bodyBytes), "&") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					postParams[parts[0]] = parts[1]
				}
			}
		}
	}

	return &HTTPRequest{
		Method:     req.Method,
		Path:       req.URL.Path,
		GetParams:  getParams,
		Headers:    headers,
		Cookies:    cookies,
		PostParams: postParams,
		Timestamp:  time.Now(),
	}, nil
}

func ParseResponse(resp *http.Response, bodyBytes []byte, requestID bson.ObjectID) (*HTTPResponse, error) {
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}

	return &HTTPResponse{
		RequestID: requestID,
		Code:      resp.StatusCode,
		Message:   http.StatusText(resp.StatusCode),
		Headers:   headers,
		Body:      string(bodyBytes),
		Timestamp: time.Now(),
	}, nil
}

func parseQueryParams(v url.Values) map[string]interface{} {
	result := make(map[string]interface{})
	for k, vals := range v {
		if len(vals) > 0 {
			result[k] = tryParseInt(vals[0])
		}
	}
	return result
}

func tryParseInt(s string) interface{} {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return s
}
