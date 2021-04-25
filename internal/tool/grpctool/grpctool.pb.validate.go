// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: internal/tool/grpctool/grpctool.proto

package grpctool

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
)

// Validate checks the field values on HttpRequest with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned. When asked to return all errors, validation continues after
// first violation, and the result is a list of violation errors wrapped in
// HttpRequestMultiError, or nil if none found. Otherwise, only the first
// error is returned, if any.
func (m *HttpRequest) Validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	switch m.Message.(type) {

	case *HttpRequest_Header_:

		if v, ok := interface{}(m.GetHeader()).(interface{ Validate(bool) error }); ok {
			if err := v.Validate(all); err != nil {
				err = HttpRequestValidationError{
					field:  "Header",
					reason: "embedded message failed validation",
					cause:  err,
				}
				if !all {
					return err
				}
				errors = append(errors, err)
			}
		}

	case *HttpRequest_Data_:

		if v, ok := interface{}(m.GetData()).(interface{ Validate(bool) error }); ok {
			if err := v.Validate(all); err != nil {
				err = HttpRequestValidationError{
					field:  "Data",
					reason: "embedded message failed validation",
					cause:  err,
				}
				if !all {
					return err
				}
				errors = append(errors, err)
			}
		}

	case *HttpRequest_Trailer_:

		if v, ok := interface{}(m.GetTrailer()).(interface{ Validate(bool) error }); ok {
			if err := v.Validate(all); err != nil {
				err = HttpRequestValidationError{
					field:  "Trailer",
					reason: "embedded message failed validation",
					cause:  err,
				}
				if !all {
					return err
				}
				errors = append(errors, err)
			}
		}

	default:
		err := HttpRequestValidationError{
			field:  "Message",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)

	}

	if len(errors) > 0 {
		return HttpRequestMultiError(errors)
	}
	return nil
}

// HttpRequestMultiError is an error wrapping multiple validation errors
// returned by HttpRequest.Validate(true) if the designated constraints aren't met.
type HttpRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpRequestMultiError) AllErrors() []error { return m }

// HttpRequestValidationError is the validation error returned by
// HttpRequest.Validate if the designated constraints aren't met.
type HttpRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpRequestValidationError) ErrorName() string { return "HttpRequestValidationError" }

// Error satisfies the builtin error interface
func (e HttpRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpRequestValidationError{}

// Validate checks the field values on HttpResponse with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned. When asked to return all errors, validation continues after
// first violation, and the result is a list of violation errors wrapped in
// HttpResponseMultiError, or nil if none found. Otherwise, only the first
// error is returned, if any.
func (m *HttpResponse) Validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	switch m.Message.(type) {

	case *HttpResponse_Header_:

		if v, ok := interface{}(m.GetHeader()).(interface{ Validate(bool) error }); ok {
			if err := v.Validate(all); err != nil {
				err = HttpResponseValidationError{
					field:  "Header",
					reason: "embedded message failed validation",
					cause:  err,
				}
				if !all {
					return err
				}
				errors = append(errors, err)
			}
		}

	case *HttpResponse_Data_:

		if v, ok := interface{}(m.GetData()).(interface{ Validate(bool) error }); ok {
			if err := v.Validate(all); err != nil {
				err = HttpResponseValidationError{
					field:  "Data",
					reason: "embedded message failed validation",
					cause:  err,
				}
				if !all {
					return err
				}
				errors = append(errors, err)
			}
		}

	case *HttpResponse_Trailer_:

		if v, ok := interface{}(m.GetTrailer()).(interface{ Validate(bool) error }); ok {
			if err := v.Validate(all); err != nil {
				err = HttpResponseValidationError{
					field:  "Trailer",
					reason: "embedded message failed validation",
					cause:  err,
				}
				if !all {
					return err
				}
				errors = append(errors, err)
			}
		}

	default:
		err := HttpResponseValidationError{
			field:  "Message",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)

	}

	if len(errors) > 0 {
		return HttpResponseMultiError(errors)
	}
	return nil
}

// HttpResponseMultiError is an error wrapping multiple validation errors
// returned by HttpResponse.Validate(true) if the designated constraints
// aren't met.
type HttpResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpResponseMultiError) AllErrors() []error { return m }

// HttpResponseValidationError is the validation error returned by
// HttpResponse.Validate if the designated constraints aren't met.
type HttpResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpResponseValidationError) ErrorName() string { return "HttpResponseValidationError" }

// Error satisfies the builtin error interface
func (e HttpResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpResponseValidationError{}

// Validate checks the field values on HttpRequest_Header with the rules
// defined in the proto definition for this message. If any rules are
// violated, an error is returned. When asked to return all errors, validation
// continues after first violation, and the result is a list of violation
// errors wrapped in HttpRequest_HeaderMultiError, or nil if none found.
// Otherwise, only the first error is returned, if any.
func (m *HttpRequest_Header) Validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if m.GetRequest() == nil {
		err := HttpRequest_HeaderValidationError{
			field:  "Request",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if v, ok := interface{}(m.GetRequest()).(interface{ Validate(bool) error }); ok {
		if err := v.Validate(all); err != nil {
			err = HttpRequest_HeaderValidationError{
				field:  "Request",
				reason: "embedded message failed validation",
				cause:  err,
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}
	}

	if v, ok := interface{}(m.GetExtra()).(interface{ Validate(bool) error }); ok {
		if err := v.Validate(all); err != nil {
			err = HttpRequest_HeaderValidationError{
				field:  "Extra",
				reason: "embedded message failed validation",
				cause:  err,
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return HttpRequest_HeaderMultiError(errors)
	}
	return nil
}

// HttpRequest_HeaderMultiError is an error wrapping multiple validation errors
// returned by HttpRequest_Header.Validate(true) if the designated constraints
// aren't met.
type HttpRequest_HeaderMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpRequest_HeaderMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpRequest_HeaderMultiError) AllErrors() []error { return m }

// HttpRequest_HeaderValidationError is the validation error returned by
// HttpRequest_Header.Validate if the designated constraints aren't met.
type HttpRequest_HeaderValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpRequest_HeaderValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpRequest_HeaderValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpRequest_HeaderValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpRequest_HeaderValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpRequest_HeaderValidationError) ErrorName() string {
	return "HttpRequest_HeaderValidationError"
}

// Error satisfies the builtin error interface
func (e HttpRequest_HeaderValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpRequest_Header.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpRequest_HeaderValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpRequest_HeaderValidationError{}

// Validate checks the field values on HttpRequest_Data with the rules defined
// in the proto definition for this message. If any rules are violated, an
// error is returned. When asked to return all errors, validation continues
// after first violation, and the result is a list of violation errors wrapped
// in HttpRequest_DataMultiError, or nil if none found. Otherwise, only the
// first error is returned, if any.
func (m *HttpRequest_Data) Validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Data

	if len(errors) > 0 {
		return HttpRequest_DataMultiError(errors)
	}
	return nil
}

// HttpRequest_DataMultiError is an error wrapping multiple validation errors
// returned by HttpRequest_Data.Validate(true) if the designated constraints
// aren't met.
type HttpRequest_DataMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpRequest_DataMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpRequest_DataMultiError) AllErrors() []error { return m }

// HttpRequest_DataValidationError is the validation error returned by
// HttpRequest_Data.Validate if the designated constraints aren't met.
type HttpRequest_DataValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpRequest_DataValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpRequest_DataValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpRequest_DataValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpRequest_DataValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpRequest_DataValidationError) ErrorName() string { return "HttpRequest_DataValidationError" }

// Error satisfies the builtin error interface
func (e HttpRequest_DataValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpRequest_Data.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpRequest_DataValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpRequest_DataValidationError{}

// Validate checks the field values on HttpRequest_Trailer with the rules
// defined in the proto definition for this message. If any rules are
// violated, an error is returned. When asked to return all errors, validation
// continues after first violation, and the result is a list of violation
// errors wrapped in HttpRequest_TrailerMultiError, or nil if none found.
// Otherwise, only the first error is returned, if any.
func (m *HttpRequest_Trailer) Validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return HttpRequest_TrailerMultiError(errors)
	}
	return nil
}

// HttpRequest_TrailerMultiError is an error wrapping multiple validation
// errors returned by HttpRequest_Trailer.Validate(true) if the designated
// constraints aren't met.
type HttpRequest_TrailerMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpRequest_TrailerMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpRequest_TrailerMultiError) AllErrors() []error { return m }

// HttpRequest_TrailerValidationError is the validation error returned by
// HttpRequest_Trailer.Validate if the designated constraints aren't met.
type HttpRequest_TrailerValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpRequest_TrailerValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpRequest_TrailerValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpRequest_TrailerValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpRequest_TrailerValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpRequest_TrailerValidationError) ErrorName() string {
	return "HttpRequest_TrailerValidationError"
}

// Error satisfies the builtin error interface
func (e HttpRequest_TrailerValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpRequest_Trailer.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpRequest_TrailerValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpRequest_TrailerValidationError{}

// Validate checks the field values on HttpResponse_Header with the rules
// defined in the proto definition for this message. If any rules are
// violated, an error is returned. When asked to return all errors, validation
// continues after first violation, and the result is a list of violation
// errors wrapped in HttpResponse_HeaderMultiError, or nil if none found.
// Otherwise, only the first error is returned, if any.
func (m *HttpResponse_Header) Validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if m.GetResponse() == nil {
		err := HttpResponse_HeaderValidationError{
			field:  "Response",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if v, ok := interface{}(m.GetResponse()).(interface{ Validate(bool) error }); ok {
		if err := v.Validate(all); err != nil {
			err = HttpResponse_HeaderValidationError{
				field:  "Response",
				reason: "embedded message failed validation",
				cause:  err,
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return HttpResponse_HeaderMultiError(errors)
	}
	return nil
}

// HttpResponse_HeaderMultiError is an error wrapping multiple validation
// errors returned by HttpResponse_Header.Validate(true) if the designated
// constraints aren't met.
type HttpResponse_HeaderMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpResponse_HeaderMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpResponse_HeaderMultiError) AllErrors() []error { return m }

// HttpResponse_HeaderValidationError is the validation error returned by
// HttpResponse_Header.Validate if the designated constraints aren't met.
type HttpResponse_HeaderValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpResponse_HeaderValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpResponse_HeaderValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpResponse_HeaderValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpResponse_HeaderValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpResponse_HeaderValidationError) ErrorName() string {
	return "HttpResponse_HeaderValidationError"
}

// Error satisfies the builtin error interface
func (e HttpResponse_HeaderValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpResponse_Header.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpResponse_HeaderValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpResponse_HeaderValidationError{}

// Validate checks the field values on HttpResponse_Data with the rules defined
// in the proto definition for this message. If any rules are violated, an
// error is returned. When asked to return all errors, validation continues
// after first violation, and the result is a list of violation errors wrapped
// in HttpResponse_DataMultiError, or nil if none found. Otherwise, only the
// first error is returned, if any.
func (m *HttpResponse_Data) Validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Data

	if len(errors) > 0 {
		return HttpResponse_DataMultiError(errors)
	}
	return nil
}

// HttpResponse_DataMultiError is an error wrapping multiple validation errors
// returned by HttpResponse_Data.Validate(true) if the designated constraints
// aren't met.
type HttpResponse_DataMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpResponse_DataMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpResponse_DataMultiError) AllErrors() []error { return m }

// HttpResponse_DataValidationError is the validation error returned by
// HttpResponse_Data.Validate if the designated constraints aren't met.
type HttpResponse_DataValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpResponse_DataValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpResponse_DataValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpResponse_DataValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpResponse_DataValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpResponse_DataValidationError) ErrorName() string {
	return "HttpResponse_DataValidationError"
}

// Error satisfies the builtin error interface
func (e HttpResponse_DataValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpResponse_Data.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpResponse_DataValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpResponse_DataValidationError{}

// Validate checks the field values on HttpResponse_Trailer with the rules
// defined in the proto definition for this message. If any rules are
// violated, an error is returned. When asked to return all errors, validation
// continues after first violation, and the result is a list of violation
// errors wrapped in HttpResponse_TrailerMultiError, or nil if none found.
// Otherwise, only the first error is returned, if any.
func (m *HttpResponse_Trailer) Validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return HttpResponse_TrailerMultiError(errors)
	}
	return nil
}

// HttpResponse_TrailerMultiError is an error wrapping multiple validation
// errors returned by HttpResponse_Trailer.Validate(true) if the designated
// constraints aren't met.
type HttpResponse_TrailerMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpResponse_TrailerMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpResponse_TrailerMultiError) AllErrors() []error { return m }

// HttpResponse_TrailerValidationError is the validation error returned by
// HttpResponse_Trailer.Validate if the designated constraints aren't met.
type HttpResponse_TrailerValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpResponse_TrailerValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpResponse_TrailerValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpResponse_TrailerValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpResponse_TrailerValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpResponse_TrailerValidationError) ErrorName() string {
	return "HttpResponse_TrailerValidationError"
}

// Error satisfies the builtin error interface
func (e HttpResponse_TrailerValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpResponse_Trailer.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpResponse_TrailerValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpResponse_TrailerValidationError{}