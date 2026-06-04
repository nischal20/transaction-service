package apperr_test

import (
	"errors"
	"testing"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/stretchr/testify/assert"
)

func TestValidation_ErrorMessage(t *testing.T) {
	err := apperr.Validation("document_number is required")
	assert.EqualError(t, err, "document_number is required")
}

func TestValidation_IsErrValidation(t *testing.T) {
	err := apperr.Validation("document_number is required")
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestValidation_IsNotOtherSentinels(t *testing.T) {
	err := apperr.Validation("document_number is required")
	assert.False(t, errors.Is(err, apperr.ErrConflict))
	assert.False(t, errors.Is(err, apperr.ErrNotFound))
}

func TestConflict_ErrorMessage(t *testing.T) {
	err := apperr.Conflict("document_number already exists")
	assert.EqualError(t, err, "document_number already exists")
}

func TestConflict_IsErrConflict(t *testing.T) {
	err := apperr.Conflict("document_number already exists")
	assert.True(t, errors.Is(err, apperr.ErrConflict))
}

func TestConflict_IsNotOtherSentinels(t *testing.T) {
	err := apperr.Conflict("document_number already exists")
	assert.False(t, errors.Is(err, apperr.ErrValidation))
	assert.False(t, errors.Is(err, apperr.ErrNotFound))
}

func TestNotFound_ErrorMessage(t *testing.T) {
	err := apperr.NotFound("account not found")
	assert.EqualError(t, err, "account not found")
}

func TestNotFound_IsErrNotFound(t *testing.T) {
	err := apperr.NotFound("account not found")
	assert.True(t, errors.Is(err, apperr.ErrNotFound))
}

func TestNotFound_IsNotOtherSentinels(t *testing.T) {
	err := apperr.NotFound("account not found")
	assert.False(t, errors.Is(err, apperr.ErrValidation))
	assert.False(t, errors.Is(err, apperr.ErrConflict))
}
