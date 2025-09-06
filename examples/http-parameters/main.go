package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/validate"
)

type CreateUser struct {
	Name string   `json:"name" form:"full_name" shape:"trim,notempty,maxlength=50" query:"name"`
	Age  int      `json:"age" shape:"min=18,max=120" form:"age" query:"age"`
	Tags []string `json:"tags" shape:"max=5,unique" form:"tags" query:"tags"`
}

type Search struct {
	Name string `json:"name" query:"q" shape:"trim,notempty" form:"name"`
	Page int    `json:"page" shape:"ifzero=1,min=1,max=1000" form:"page" query:"page"`
}

var searchSchema = shape.FromTags[Search]()

func createUser(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		writeError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var values url.Values
	switch mediaType {
	case "application/x-www-form-urlencoded":
		err = r.ParseForm()
		values = r.PostForm
	case "multipart/form-data":
		err = r.ParseMultipartForm(64 << 10)
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
			values = r.MultipartForm.Value
		}
	default:
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	var request CreateUser
	if err := shape.BindFormContext(r.Context(), &request, values, shape.FormOptions{DisallowUnknownFields: true}); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, request)
}

func createUserFromRequest(w http.ResponseWriter, r *http.Request) {
	var request CreateUser
	if err := shape.BindRequest(&request, r, shape.RequestOptions{DisallowUnknownFields: true}); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, request)
}

func searchUsers(w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeError(w, err)
		return
	}
	request, err := searchSchema.ParseQueryContext(r.Context(), values, shape.QueryOptions{DisallowUnknownFields: true})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	var tooLarge *http.MaxBytesError
	var validation *validate.Error
	switch {
	case errors.Is(err, shape.ErrRequestTooLarge), errors.As(err, &tooLarge):
		status = http.StatusRequestEntityTooLarge
	case errors.Is(err, shape.ErrUnsupportedContentType):
		status = http.StatusUnsupportedMediaType
	case errors.As(err, &validation):
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func main() {
	automatic := httptest.NewRequest(http.MethodPost, "/users?age=20", strings.NewReader(`{"name":" Pong "}`))
	automatic.Header.Set("Content-Type", "application/json")
	run(createUserFromRequest, automatic)

	form := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("full_name=+Pong+&age=20&tags=go&tags=web"))
	form.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	run(createUser, form)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("full_name", " Pong "); err != nil {
		panic(err)
	}
	if err := writer.WriteField("age", "20"); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	multi := httptest.NewRequest(http.MethodPost, "/users", &body)
	multi.Header.Set("Content-Type", writer.FormDataContentType())
	run(createUser, multi)
	run(searchUsers, httptest.NewRequest(http.MethodGet, "/users?q=+Pong+&page=2", nil))
}

func run(handler http.HandlerFunc, request *http.Request) {
	response := httptest.NewRecorder()
	handler(response, request)
	fmt.Printf("status=%d body=%s", response.Code, response.Body.String())
}
