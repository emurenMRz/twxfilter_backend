package mediadata

import (
	"fmt"
	"net/http"
)

type MediaError struct {
	code    int
	message string
}

func (e *MediaError) Error() string {
	return e.message
}

func (e *MediaError) IsNotFound() bool {
	return e.code == http.StatusNotFound
}

func MediaInternalServerError(format string, a ...any) error {
	return &MediaError{code: http.StatusInternalServerError, message: fmt.Sprintf(format, a...)}
}

func MediaNotFoundError(format string, a ...any) error {
	return &MediaError{code: http.StatusNotFound, message: fmt.Sprintf(format, a...)}
}

func MediaBadGatewayError(format string, a ...any) error {
	return &MediaError{code: http.StatusBadGateway, message: fmt.Sprintf(format, a...)}
}

func MediaNoContentError(format string, a ...any) error {
	return &MediaError{code: http.StatusNoContent, message: fmt.Sprintf(format, a...)}
}
