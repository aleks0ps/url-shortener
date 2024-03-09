package http

import (
	"net/http"
	"strconv"
)

// Content types
type ContentType int

const (
	None ContentType = iota
	PlainText
	URLEncoded
	JSON
	JS
	CSS
	HTML
	XML
)

type ContentTypes struct {
	Name string
	Code ContentType
}

var SupportedTypes = []ContentTypes{
	{
		Name: "text/plain",
		Code: PlainText,
	},
	{
		Name: "application/x-www-form-urlencoded",
		Code: URLEncoded,
	},
	{
		Name: "application/json",
		Code: JSON,
	},
	{
		Name: "application/javascript",
		Code: JS,
	},
	{
		Name: "text/css",
		Code: CSS,
	},
	{
		Name: "text/html",
		Code: HTML,
	},
	{
		Name: "text/xml",
		Code: XML,
	},
}

func GetContentTypeCode(name string) ContentType {
	for _, t := range SupportedTypes {
		if name == t.Name {
			return t.Code
		}
	}
	return None
}

func GetContentTypeName(code ContentType) string {
	for _, t := range SupportedTypes {
		if code == t.Code {
			return t.Name
		}
	}
	return "none"
}

func WriteResponse(w *http.ResponseWriter, t ContentType, status int, data []byte) {
	switch t {
	case None:
		(*w).WriteHeader(status)
		if data != nil {
			(*w).Write(data)
		}
	default:
		(*w).Header().Set("Content-Type", GetContentTypeName(t))
		(*w).Header().Set("Content-Length", strconv.Itoa(len(data)))
		(*w).WriteHeader(status)
		(*w).Write(data)
	}
}

func WriteError(w *http.ResponseWriter, status int, err error) {
	http.Error(*w, err.Error(), status)
}
