package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewError(t *testing.T) {
	e := NewError(http.StatusInternalServerError)
	require.Error(t, e)
	assert.Equal(t, &Error{StatusCode: http.StatusInternalServerError}, e)

	wantStatus := http.StatusText(http.StatusInternalServerError)
	assert.Equal(t, wantStatus, e.Error())
	assert.Equal(t, "500 "+wantStatus, e.String())
	require.NoError(t, e.Unwrap())

	b, err := e.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"error_message": "`+e.Error()+`"}`, string(b))

	e = NewError(http.StatusInternalServerError).
		WithStatus("oops something wrong")
	require.Error(t, e)

	assert.Equal(t, &Error{
		Status:     "oops something wrong",
		StatusCode: http.StatusInternalServerError,
	}, e)
	assert.Equal(t, e.Status, e.Error())
	assert.Equal(t, "500 "+e.Status, e.String())

	b, err = e.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"error_message": "`+e.Error()+`"}`, string(b))
}

func TestWrapError(t *testing.T) {
	wantErr := errors.New("something wrong")
	wrappedErr := WrapServerError(wantErr)
	require.Error(t, wrappedErr)
	require.ErrorIs(t, wrappedErr, wantErr)
	assert.Same(t, wrappedErr, WrapServerError(wrappedErr))

	assert.Equal(t, wantErr.Error(), wrappedErr.Error())
	assert.Equal(t,
		"500 "+http.StatusText(http.StatusInternalServerError)+": "+wantErr.Error(),
		wrappedErr.String())
	assert.Same(t, wantErr, wrappedErr.Unwrap())

	b, err := wrappedErr.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"error_message": "`+wantErr.Error()+`"}`, string(b))

	wrappedErr = WrapServerError(wantErr).WithStatus("Custom Server Error")
	require.Error(t, wrappedErr)
	assert.Equal(t, wantErr.Error(), wrappedErr.Error())
	assert.Equal(t,
		"500 "+wrappedErr.Status+": "+wantErr.Error(),
		wrappedErr.String())
	assert.Same(t, wantErr, wrappedErr.Unwrap())

	b, err = wrappedErr.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"error_message": "`+wantErr.Error()+`"}`, string(b))
}
