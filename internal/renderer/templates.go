package renderer

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"text/template"
)

//go:embed templates/*.gotmpl
var templatesFS embed.FS

var templates = func() *template.Template {
	templates, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic("can't create embed templates fs: " + err.Error())
	}

	funcMap := template.FuncMap{
		"FormatRate": func(value float64) string {
			return fmt.Sprintf("%.3f", value)
		},
	}

	return template.Must(
		template.New("").Funcs(funcMap).
			ParseFS(templates, "*.gotmpl"),
	)
}()

func renderTemplate(w io.Writer, name string, data interface{}) error {
	return templates.ExecuteTemplate(w, name, data)
}
