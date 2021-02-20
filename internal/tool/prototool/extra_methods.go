package prototool

import (
	"net/http"
	"net/url"
)

func (x *HttpRequest) HttpHeader() http.Header {
	return ValuesMapToHttpHeader(x.Header)
}

func (x *HttpResponse) HttpHeader() http.Header {
	return ValuesMapToHttpHeader(x.Header)
}

func (x *HttpRequest) UrlQuery() url.Values {
	return UrlValuesFromValuesMap(x.Query)
}

func ValuesMapToHttpHeader(from map[string]*Values) http.Header {
	res := make(http.Header, len(from))
	for key, val := range from {
		res[key] = val.Value
	}
	return res
}

func ValuesMapFromHttpHeader(from http.Header) map[string]*Values {
	res := make(map[string]*Values, len(from))
	for key, val := range from {
		res[key] = &Values{
			Value: val,
		}
	}
	return res
}

func ValuesMapFromUrlValues(from url.Values) map[string]*Values {
	res := make(map[string]*Values, len(from))
	for key, val := range from {
		res[key] = &Values{
			Value: val,
		}
	}
	return res
}

func UrlValuesFromValuesMap(from map[string]*Values) url.Values {
	query := make(url.Values, len(from))
	for key, val := range from {
		query[key] = val.Value
	}
	return query
}
