package testhelpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Validatable interface {
	Validate() error
}

type InvalidTestcase struct {
	Name      string
	ErrString string
	Invalid   Validatable
}

type ValidTestcase struct {
	Name  string
	Valid Validatable
}

func AssertInvalid(t *testing.T, tests []InvalidTestcase) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) { // nolint: scopelint
			t.Helper()
			err := tc.Invalid.Validate()            // nolint: scopelint
			assert.EqualError(t, err, tc.ErrString) // nolint: scopelint
		})
	}
}

func AssertValid(t *testing.T, tests []ValidTestcase) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) { // nolint: scopelint
			t.Helper()
			assert.NoError(t, tc.Valid.Validate()) // nolint: scopelint
		})
	}
}
