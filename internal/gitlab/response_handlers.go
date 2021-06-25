package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
)

type ResponseHandlerStruct struct {
	AcceptHeader string
	HandleFunc   func(*http.Response, error) error
}

func (r ResponseHandlerStruct) Handle(resp *http.Response, err error) error {
	return r.HandleFunc(resp, err)
}

func (r ResponseHandlerStruct) Accept() string {
	return r.AcceptHeader
}

func NakedResponseHandler(response **http.Response) ResponseHandler {
	return ResponseHandlerStruct{
		HandleFunc: func(r *http.Response, err error) error {
			if err != nil {
				return err
			}
			*response = r
			return nil
		},
	}
}

func JsonResponseHandler(response interface{}) ResponseHandler {
	return ResponseHandlerStruct{
		AcceptHeader: "application/json",
		HandleFunc: func(resp *http.Response, err error) (retErr error) {
			if err != nil {
				return err
			}
			defer errz.SafeClose(resp.Body, &retErr)
			switch resp.StatusCode {
			case http.StatusOK:
				if !isApplicationJSON(resp) {
					return fmt.Errorf("unexpected Content-Type in response: %q", resp.Header.Get("Content-Type"))
				}
				data, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("response body read: %v", err)
				}
				if err = json.Unmarshal(data, response); err != nil {
					return fmt.Errorf("WithJsonResponseHandler: json.Unmarshal: %v", err)
				}
				return nil
			case http.StatusUnauthorized: // No token, invalid token, revoked token
				return &ClientError{
					Kind:       ErrorKindUnauthorized,
					StatusCode: http.StatusUnauthorized,
				}
			case http.StatusForbidden: // Access denied
				return &ClientError{
					Kind:       ErrorKindForbidden,
					StatusCode: http.StatusForbidden,
				}
			default: // Unexpected status
				return &ClientError{
					Kind:       ErrorKindOther,
					StatusCode: resp.StatusCode,
				}
			}
		},
	}
}

// NoContentResponseHandler can be used when no response is expected or response must be discarded.
func NoContentResponseHandler() ResponseHandler {
	return ResponseHandlerStruct{
		HandleFunc: func(resp *http.Response, err error) (retErr error) {
			if err != nil {
				return err
			}
			defer errz.SafeClose(resp.Body, &retErr)
			switch resp.StatusCode {
			case http.StatusOK, http.StatusNoContent:
				const maxBodySlurpSize = 8 * 1024
				_, _ = io.CopyN(io.Discard, resp.Body, maxBodySlurpSize)
				return nil
			case http.StatusUnauthorized: // No token, invalid token, revoked token
				return &ClientError{
					Kind:       ErrorKindUnauthorized,
					StatusCode: http.StatusUnauthorized,
				}
			case http.StatusForbidden: // Access denied
				return &ClientError{
					Kind:       ErrorKindForbidden,
					StatusCode: http.StatusForbidden,
				}
			default: // Unexpected status
				return &ClientError{
					Kind:       ErrorKindOther,
					StatusCode: resp.StatusCode,
				}
			}
		},
	}
}

func isContentType(expected, actual string) bool {
	parsed, _, err := mime.ParseMediaType(actual)
	return err == nil && parsed == expected
}

func isApplicationJSON(r *http.Response) bool {
	contentType := r.Header.Get("Content-Type")
	return isContentType("application/json", contentType)
}
