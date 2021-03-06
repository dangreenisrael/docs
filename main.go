package main

import (
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/convox/docs/pkg/docs"
	"github.com/convox/stdapi"
)

var categorySlugs = []string{
	"introduction",
	"deployment",
	"development",
	"management",
	"monitoring",
	"resources",
	"integrations",
	"console",
	"reference",
	"gen1",
	"help",
}

func init() {
	if err := docs.LoadCategories(categorySlugs...); err != nil {
		log.Fatal(err)
	}

	if err := docs.UploadIndex(); err != nil {
		log.Printf("error: %s", err)
	}
}

func main() {
	s := stdapi.New("docs", "docs.convox")

	s.Use(func(fn stdapi.HandlerFunc) stdapi.HandlerFunc {
		return func(c *stdapi.Context) error {
			if c.Request().Header.Get("X-Forwarded-Proto") == "http" {
				fmt.Printf("c.Request().Host = %+v\n", c.Request().Host)
				u := *(c.Request().URL)
				u.Host = c.Request().Host
				u.Scheme = "https"
				return c.Redirect(301, u.String())
			}
			return fn(c)
		}
	})

	s.Router.Static("/assets/", "./assets")

	s.Route("GET", "/", index)
	s.Route("GET", "/docs/{slug}", redirect)
	s.Route("GET", "/docs/{slug}/", redirect)
	s.Route("GET", "/{category}/{slug}", doc)
	s.Route("GET", "/{slug}", redirect)
	s.Route("GET", "/{slug}/", redirect)

	stdapi.LoadTemplates("./templates", helpers)

	if err := s.Listen("https", ":3000"); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

func helpers(c *stdapi.Context) template.FuncMap {
	return template.FuncMap{
		"env": func(s string) string {
			return os.Getenv(s)
		},
	}
}

func index(c *stdapi.Context) error {
	return c.Redirect(302, "/introduction/getting-started")
}

func doc(c *stdapi.Context) error {
	params := map[string]interface{}{
		"Categories": docs.CategoryList(),
		"Slug":       "",
	}

	cc, ok := docs.CategoryList().Find(c.Var("category"))
	if !ok {
		return c.RenderTemplate("404", params)
	}

	params["Category"] = cc.Slug

	d, ok := cc.Documents.Find(c.Var("slug"))
	if !ok {
		return c.RenderTemplate("404", params)
	}

	params["Document"] = template.HTML(d.Body)
	params["Slug"] = d.Slug
	params["Title"] = d.Title

	return c.RenderTemplate("doc", params)
}

func redirect(c *stdapi.Context) error {
	for _, cc := range docs.CategoryList() {
		if d, ok := cc.Documents.Find(c.Var("slug")); ok {
			return c.Redirect(301, fmt.Sprintf("/%s/%s", cc.Slug, d.Slug))
		}
	}

	return stdapi.Errorf(404, "not found")
}
