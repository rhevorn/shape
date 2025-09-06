package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestFormBodyDoesNotMergeQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/users?age=1&extra=ignored", strings.NewReader("full_name=+Pong+&age=20&tags=go&tags=web"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	createUser(response, request)
	var got CreateUser
	if response.Code != http.StatusCreated {
		t.Fatal(response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "Pong" || got.Age != 20 || !reflect.DeepEqual(got.Tags, []string{"go", "web"}) {
		t.Fatalf("got=%#v", got)
	}
}

func TestMultipartTextFields(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, field := range [][2]string{{"full_name", " Pong "}, {"age", "20"}, {"tags", "go"}, {"tags", "web"}} {
		if err := writer.WriteField(field[0], field[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/users?age=1", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	createUser(response, request)
	if response.Code != http.StatusCreated {
		t.Fatal(response.Code, response.Body.String())
	}
	var got CreateUser
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Age != 20 || got.Name != "Pong" || !reflect.DeepEqual(got.Tags, []string{"go", "web"}) {
		t.Fatalf("got=%#v", got)
	}
}

func TestFormHTTPFailures(t *testing.T) {
	cases := []struct {
		body   string
		status int
	}{
		{"full_name=Pong&age=20&age=21", 400},
		{"full_name=Pong&age=1", 422},
		{"full_name=Pong&age=20&unknown=x", 400},
		{"full_name=%GG&age=20", 400},
		{"full_name=" + strings.Repeat("x", 1<<20), 413},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		createUser(response, request)
		if response.Code != tc.status {
			t.Fatalf("status=%d want=%d body=%s", response.Code, tc.status, response.Body.String())
		}
	}
}

func TestQueryHTTP(t *testing.T) {
	for _, tc := range []struct {
		query  string
		status int
	}{
		{"q=+Pong+", 200}, {"q=Pong&page=2", 200},
		{"q=Pong&page=2&page=3", 400}, {"q=Pong&page=bad", 400},
		{"q=%GG", 400}, {"q=Pong;page=2", 400},
		{"q=Pong&extra=x", 400}, {"q=+", 422},
	} {
		request := httptest.NewRequest(http.MethodGet, "/users?"+tc.query, nil)
		response := httptest.NewRecorder()
		searchUsers(response, request)
		if response.Code != tc.status {
			t.Fatalf("query=%s status=%d body=%s", tc.query, response.Code, response.Body.String())
		}
		if tc.status == 200 {
			var got Search
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Name != "Pong" || got.Page < 1 {
				t.Fatalf("got=%#v", got)
			}
		}
	}
}

func TestAutomaticRequestBinding(t *testing.T) {
	for _, tc := range []struct {
		query, body string
		status      int
	}{
		{"age=20", `{"name":" Pong "}`, http.StatusCreated},
		{"age=20", `{"name":"Pong","age":20}`, http.StatusBadRequest},
		{"age=1", `{"name":"Pong"}`, http.StatusUnprocessableEntity},
	} {
		request := httptest.NewRequest(http.MethodPost, "/users?"+tc.query, strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		createUserFromRequest(response, request)
		if response.Code != tc.status {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	}
}
