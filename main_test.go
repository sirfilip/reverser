package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Subject() *App {
	templates := template.Must(template.ParseGlob("templates/*.html"))
	store := NewStore()
	app := NewApp(templates, store)
	app.Setup()
	return app
}

func TestProxyPass(t *testing.T) {
	getData := map[string]string{
		"/proxy/testing":               "/, GET",
		"/proxy/testing/one/two/three": "/one/two/three, GET",
		"/proxy/testing/?foo=bar":      "/?foo=bar, GET",
	}
	postData := map[string]string{
		"/proxy/testing":               "/, POST",
		"/proxy/testing/one/two/three": "/one/two/three, POST",
		"/proxy/testing/?foo=bar":      "/?foo=bar, POST",
	}

	app := Subject()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%s, %s", r.URL, r.Method)))
	}))
	defer server.Close()
	app.Register(server.URL, "testing")
	proxy := httptest.NewServer(app.Router)
	defer server.Close()

	for path, expected := range getData {

		res, err := http.Get(proxy.URL + path)
		if err != nil {
			t.Errorf("Unexpected error %s", err)
		}
		content, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Errorf("Unexpected error %s", err)
		}
		if string(content) != expected {
			t.Errorf("Expected %s, got %s", expected, string(content))
		}
	}
	for path, expected := range postData {

		res, err := http.Post(proxy.URL+path, "", nil)
		if err != nil {
			t.Errorf("Unexpected error %s", err)
		}
		content, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Errorf("Unexpected error %s", err)
		}
		if string(content) != expected {
			t.Errorf("Expected %s, got %s", expected, string(content))
		}
	}
}
