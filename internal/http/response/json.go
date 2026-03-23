package response

import (
	"encoding/json"
	"net/http"
)

const applicationJSON = "application/json"

// JSON creates a new JSON response with a 200 status code.
func JSON[T any](handler func(http.ResponseWriter, *http.Request) (T, error),
	opts ...Option,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := handler(w, r)
		if err != nil {
			ServerErrorJSON(w, r, err)
			return
		}
		MarshalJSON(w, r, v, opts...)
	}
}

func MarshalJSON(w http.ResponseWriter, r *http.Request, v any,
	opts ...Option,
) {
	b, err := json.Marshal(v)
	if err != nil {
		ServerErrorJSON(w, r, err)
		return
	}

	New(w, r, opts...).
		WithHeader(contentType, applicationJSON).
		WithBodyAsBytes(b).
		Write()
}

// CreatedJSON sends a created response to the client as JSON.
func CreatedJSON[T any](
	handler func(http.ResponseWriter, *http.Request) (T, error),
) http.HandlerFunc {
	return JSON(handler, WithStatusCreated())
}

func AcceptedJSON(handler func(http.ResponseWriter, *http.Request) error,
) http.HandlerFunc {
	return WithStatusJSON(handler, http.StatusAccepted)
}

func WithStatusJSON(handler func(http.ResponseWriter, *http.Request) error,
	statusCode int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err != nil {
			ServerErrorJSON(w, r, err)
			return
		}
		New(w, r).WithStatus(statusCode).Write()
	}
}

func NoContentJSON(handler func(http.ResponseWriter, *http.Request) error,
) http.HandlerFunc {
	return WithStatusJSON(handler, http.StatusNoContent)
}

// UnauthorizedJSON sends a not authorized error to the client as JSON.
func UnauthorizedJSON(w http.ResponseWriter, r *http.Request) {
	ErrUnauthorized.ServeJSON(w, r)
}

// ServerErrorJSON sends an internal error to the client as JSON.
func ServerErrorJSON(w http.ResponseWriter, r *http.Request, err error) {
	WrapServerError(err).ServeJSON(w, r)
}

func BadRequestJSON(w http.ResponseWriter, r *http.Request, err error) {
	WrapBadRequest(err).ServeJSON(w, r)
}

func NotFoundJSON(w http.ResponseWriter, r *http.Request) {
	ErrNotFound.ServeJSON(w, r)
}
