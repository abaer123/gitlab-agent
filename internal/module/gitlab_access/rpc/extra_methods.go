package rpc

import (
	"net/http"
	"net/url"
)

func (x *Response_Header) ToHttpHeader() http.Header {
	return toHttpHeader(x.Header)
}

func (x *Request_Header) ToHttpHeader() http.Header {
	return toHttpHeader(x.Header)
}

func (x *Request_Header) ToUrlQuery() url.Values {
	query := make(url.Values, len(x.Query))
	for key, vals := range x.Query {
		query[key] = vals.Value
	}
	return query
}

func toHttpHeader(from map[string]*Values) http.Header {
	res := make(http.Header, len(from))
	for key, val := range from {
		res[key] = val.Value
	}
	return res
}

func HeaderFromHttpHeader(from http.Header) map[string]*Values {
	res := make(map[string]*Values, len(from))
	for key, val := range from {
		res[key] = &Values{
			Value: val,
		}
	}
	return res
}

func QueryFromUrlValues(from url.Values) map[string]*Values {
	res := make(map[string]*Values, len(from))
	for key, val := range from {
		res[key] = &Values{
			Value: val,
		}
	}
	return res
}
