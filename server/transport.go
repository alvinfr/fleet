package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/datastore"
	"golang.org/x/net/context"
)

var (
	// errBadRoute is used for mux errors
	errBadRoute = errors.New("bad route")
)

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}

	if e, ok := response.(statuser); ok {
		w.WriteHeader(e.status())
		if e.status() == http.StatusNoContent {
			return nil
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}

// erroer interface is implemented by response structs to encode business logic errors
type errorer interface {
	error() error
}

// statuser allows response types to implement a custom
// http success status - default is 200 OK
type statuser interface {
	status() int
}

// encode errors from business-logic
func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch err {
	case datastore.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	case datastore.ErrExists:
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(typeErrsStatus(err))
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	// decide on error type and encode proper JSON format
	type validator interface {
		Invalid() []map[string]string
	}
	if e, ok := err.(validator); ok {
		var ve = struct {
			Message string              `json:"message"`
			Errors  []map[string]string `json:"errors"`
		}{
			Message: "Validation Failed",
			Errors:  e.Invalid(),
		}
		enc.Encode(ve)
		return
	}

	// other errors
	enc.Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func typeErrsStatus(err error) int {
	switch err.(type) {
	case invalidArgumentError:
		return http.StatusUnprocessableEntity
	case authError:
		return http.StatusUnauthorized
	case forbiddenError:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func idFromRequest(r *http.Request, name string) (uint, error) {
	vars := mux.Vars(r)
	id, ok := vars[name]
	if !ok {
		return 0, errBadRoute
	}
	uid, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}
	return uint(uid), nil
}

func decodeNoParamsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}