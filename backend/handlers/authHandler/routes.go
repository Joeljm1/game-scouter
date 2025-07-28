package auth

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed templates
var FS embed.FS

var welcomeTmpl *template.Template

func init() {
	tmpl, err := template.New("welcome").ParseFS(FS, "templates/Welcome.tmpl")
	if err != nil {
		panic(err)
	}
	welcomeTmpl = tmpl
}

func (app *AuthApplication) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", app.RegisterUserHandler)
	r.Post("/login", app.LoginHandler)
	r.Get("/activate", app.ActivateUserHandler)
	return r
}
