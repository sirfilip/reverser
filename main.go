package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

type DataStore interface {
	Register(string, string) error
	Unregister(string) error
	ProxyList() map[string]*Proxy
	Find(string) (*Proxy, error)
}

type Proxy struct {
	Path string
	URL  *url.URL
}

func (p *Proxy) Handler() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Host = p.URL.Host
			req.URL.Scheme = p.URL.Scheme
			req.URL.Host = p.URL.Host
		},
	}
}

type Store struct {
	sync.Mutex
	store map[string]*Proxy
}

func (s *Store) Register(target string, path string) error {
	s.Lock()
	defer s.Unlock()
	targetURL, err := url.Parse(target)
	if err != nil {
		return err
	}
	s.store[path] = &Proxy{
		Path: path,
		URL:  targetURL,
	}
	return nil
}

func (s *Store) Unregister(path string) error {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.store[path]; !ok {
		return errors.New(fmt.Sprintf("Path %s is not registered", path))
	}
	delete(s.store, path)
	return nil
}

func (s *Store) Find(path string) (*Proxy, error) {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.store[path]; !ok {
		return nil, errors.New(fmt.Sprintf("Path %s not found", path))
	}
	return s.store[path], nil
}

func (s *Store) ProxyList() map[string]*Proxy {
	s.Lock()
	defer s.Unlock()
	result := make(map[string]*Proxy)
	for k, v := range s.store {
		result[k] = v
	}
	return result
}

func NewStore() *Store {
	return &Store{store: make(map[string]*Proxy)}
}

type AppInterface interface {
	DataStore
	ExecuteTemplate(io.Writer, string, interface{}) error
}
type App struct {
	DataStore
	Router   *mux.Router
	Template *template.Template
}

func (app *App) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	return app.Template.ExecuteTemplate(w, name, data)
}

type RouteHandler func(AppInterface) http.HandlerFunc

func (app *App) RegisterHandler(path string, handler RouteHandler) {
	app.Router.HandleFunc(path, handler(app))
}

func (app *App) MountProxyHandler() {

	app.Router.PathPrefix("/proxy/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		proxyId := parts[2]
		proxy, err := app.Find(proxyId)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.StripPrefix(fmt.Sprintf("/proxy/%s", proxyId), proxy.Handler()).ServeHTTP(w, r)
	})
}

func NewApp(template *template.Template, store DataStore) *App {
	router := mux.NewRouter()
	return &App{Router: router, Template: template, DataStore: store}
}

func NewViewContext() map[string]interface{} {
	return make(map[string]interface{})
}

type Formable interface {
	Errors() map[string]string
	Submit(*http.Request) bool
	Value(string) string
}

type RegisterForm struct {
	store  DataStore
	errors map[string]string
	values map[string]string
}

func (rf *RegisterForm) Errors() map[string]string {
	return rf.errors
}

func (rf *RegisterForm) Submit(r *http.Request) bool {
	if r.Method != "POST" {
		return false
	}

	rf.values["Path"] = r.FormValue("path")
	rf.values["Target"] = r.FormValue("target")

	if !rf.Valid() {
		return false
	}

	rf.store.Register(rf.Value("Target"), rf.Value("Path"))
	return true
}

func (rf *RegisterForm) Valid() bool {
	rf.errors = make(map[string]string)

	if rf.Value("Path") == "" {
		rf.errors["Path"] = "Path is required"
		return false
	}

	if rf.Value("Target") == "" {
		rf.errors["Target"] = "The target url is required"
		return false
	}

	if _, err := url.Parse(rf.Value("Target")); err != nil {
		rf.errors["Target"] = err.Error()
		return false
	}

	return true
}

func (rf *RegisterForm) Values() map[string]string {
	return rf.values
}

func (rf *RegisterForm) Value(val string) string {
	return rf.values[val]
}

func NewRegisterForm(store DataStore) *RegisterForm {
	return &RegisterForm{errors: make(map[string]string), values: make(map[string]string), store: store}
}

func (app *App) Setup() {
	app.RegisterHandler("/", func(app AppInterface) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			viewContext := NewViewContext()
			viewContext["ProxyList"] = app.ProxyList()
			viewContext["Title"] = "reverser-home"
			app.ExecuteTemplate(w, "index.html", viewContext)
		}
	})
	app.RegisterHandler("/register", func(app AppInterface) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			viewContext := NewViewContext()
			form := NewRegisterForm(app)
			if form.Submit(r) {
				http.Redirect(w, r, "/", 302)
				return
			}
			viewContext["Form"] = form
			viewContext["Title"] = "reverser-add"
			app.ExecuteTemplate(w, "register.html", viewContext)
		}
	})
	app.RegisterHandler("/unregister", func(app AppInterface) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			app.Unregister(r.URL.Query().Get("path"))
			http.Redirect(w, r, "/", 302)
		}
	})

	app.MountProxyHandler()

}

func main() {
	templates := template.Must(template.ParseGlob("templates/*.html"))
	store := NewStore()
	app := NewApp(templates, store)
	app.Setup()
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.Handle("/", app.Router)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
