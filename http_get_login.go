package crudui

import (
	"bytes"
	"embed"
	"log"
	"net/http"
	"text/template"
)

func (c *Controller) tryGetLogin(w http.ResponseWriter, r *http.Request, uri string) bool {
	realURI := c.getRealURI(uri, r.RequestURI)
	if realURI == "login/" {
		c.renderLogin(w, r, uri)
		return true
	}
	return false
}

func (c *Controller) renderLogin(w http.ResponseWriter, r *http.Request, uri string) {
	configCss, _ := embed.FS.ReadFile(htmlDir, "html/config.css")
	stylesCss, _ := embed.FS.ReadFile(htmlDir, "html/styles.css")

	loginTpl, _ := embed.FS.ReadFile(htmlDir, "html/login.html")

	tplObj := struct {
		URI       string
		ConfigCss string
		StylesCss string
	}{
		URI:       uri,
		ConfigCss: string(configCss),
		StylesCss: string(stylesCss),
	}
	buf := &bytes.Buffer{}
	t := template.Must(template.New("index").Parse(string(loginTpl)))
	err := t.Execute(buf, &tplObj)
	if err != nil {
		log.Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(buf.Bytes())
}
