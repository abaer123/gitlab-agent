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
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Invalid.Validate()
			assert.EqualError(t, err, tc.ErrString)
		})
	}
}

func AssertValid(t *testing.T, tests []ValidTestcase) {
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			assert.NoError(t, tc.Valid.Validate())
		})
	}
}
