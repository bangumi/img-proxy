package main

import (
	"embed"
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

//go:embed templates
var readmeTemplate embed.FS

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newRender() echo.Renderer {
	return &Template{
		templates: template.Must(template.New("").ParseFS(readmeTemplate, "templates/*.gohtml")),
	}
}
