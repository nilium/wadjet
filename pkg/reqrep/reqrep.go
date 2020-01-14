package reqrep

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// HTTP response writing.

func JSON(w http.ResponseWriter, code int, msg interface{}) error {
	p, err := json.Marshal(msg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("unable to encode response %T: %w", msg, err)
	}

	// Set content type and length.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(p)))
	w.WriteHeader(code)

	_, err = w.Write(p)
	if err != nil {
		return fmt.Errorf("unable to write response: %w", err)
	}

	return nil
}

func Errorf(w http.ResponseWriter, code int, format string, args ...interface{}) error {
	return Error(w, code, fmt.Sprintf(format, args...))
}

func Error(w http.ResponseWriter, code int, msg string) error {
	resp := struct {
		Code    int    `json:"code"`
		Message string `json:"error"`
	}{
		Code:    code,
		Message: msg,
	}
	return JSON(w, code, resp)
}

func Code(w http.ResponseWriter, code int, msg string) error {
	resp := struct {
		Code    int    `json:"code"`
		Message string `json:"message,omitempty"`
	}{
		Code:    code,
		Message: msg,
	}
	return JSON(w, code, resp)
}

// Error handling.

func AllowMethods(methods ...string) func(http.Handler) http.HandlerFunc {
	var allowed []string
	want := map[string]struct{}{}
	for _, method := range methods {
		method = strings.ToUpper(method)
		if _, ok := want[method]; !ok {
			want[method] = struct{}{}
			allowed = append(allowed, method)
		}
	}
	sort.Strings(allowed)

	invalidate := func(w http.ResponseWriter, req *http.Request) bool {
		if _, ok := want[req.Method]; ok {
			return false
		}
		for _, method := range allowed {
			w.Header().Add("Allow", method)
		}
		Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return true
	}

	return func(next http.Handler) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if invalidate(w, req) {
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

var (
	AllowGET    = AllowMethods("GET", "HEAD")
	AllowPOST   = AllowMethods("POST")
	AllowCommon = AllowMethods("GET", "HEAD", "POST")
)

func Accept(w http.ResponseWriter, accept ...string) {
	for _, typ := range accept {
		w.Header().Add("Accept", typ)
	}
	_ = Error(w, http.StatusNotAcceptable, "unacceptable content type")
}

type HTTPError struct {
	Code int
	Err  string
}

func (e *HTTPError) Error() string {
	return e.Err
}

func TrapError(w http.ResponseWriter) {
	switch err := recover().(type) {
	case nil:
		return
	case *HTTPError:
		Error(w, err.Code, err.Err)
	case error:
		Error(w, http.StatusInternalServerError, "unexpected error: "+err.Error())
	}
}

func NewHTTPError(code int, format string, args ...interface{}) error {
	return &HTTPError{
		Code: code,
		Err:  fmt.Sprintf(format, args...),
	}
}

func Bail(code int, format string, args ...interface{}) {
	panic(NewHTTPError(code, format, args...))
}
