package utils_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nischalpatel/transactions-api/internal/utils"
	"github.com/stretchr/testify/assert"
)

type unserializable struct{}

func (unserializable) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("cannot marshal")
}

func TestWriteJSON_EncodeError(t *testing.T) {
	rec := httptest.NewRecorder()
	utils.WriteJSON(rec, http.StatusOK, unserializable{})
	assert.Equal(t, http.StatusOK, rec.Code)
}
