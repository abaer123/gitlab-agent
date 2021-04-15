package httpz

import (
	"net/http"
	"net/textproto"
	"strings"
)

// RemoveConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See https://tools.ietf.org/html/rfc7230#section-6.1
func RemoveConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}
