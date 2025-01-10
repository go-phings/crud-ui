package crudui

import (
	"bytes"
	"embed"
	"log"
	"net/http"
	"text/template"
)

func (c *Controller) tryGetHome(w http.ResponseWriter, r *http.Request, uri string, objFuncs ...func() interface{}) bool {
	realURI := c.getRealURI(uri, r.RequestURI)
	if realURI == "" {
		c.renderMain(w, r, uri, objFuncs...)
		return true
	}
	return false
}

func (c *Controller) renderMain(w http.ResponseWriter, r *http.Request, uri string, objFuncs ...func() interface{}) {
	configCss, _ := embed.FS.ReadFile(htmlDir, "html/config.css")
	stylesCss, _ := embed.FS.ReadFile(htmlDir, "html/styles.css")

	indexTpl, _ := embed.FS.ReadFile(htmlDir, "html/index.html")

	structListTpl, err := c.getStructListHTML(uri, r, objFuncs...)
	if err != nil {
		log.Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	contentHomeTpl, _ := embed.FS.ReadFile(htmlDir, "html/content_home.html")

	ctxId := r.Context().Value(ContextValue("LoggedUserID"))
	ctxName := r.Context().Value(ContextValue("LoggedUserName"))

	userId := "0"
	if ctxId != nil {
		userId = ctxId.(string)
	}
	userName := ""
	if ctxName != nil {
		userName = ctxName.(string)
	}

	tplObj := struct {
		URI        string
		StructList string
		Content    string
		ConfigCss  string
		StylesCss  string
		Username   string
		UserID     string
	}{
		URI:        uri,
		StructList: structListTpl,
		Content:    string(contentHomeTpl),
		ConfigCss:  string(configCss),
		StylesCss:  string(stylesCss),
		Username:   userName,
		UserID:     userId,
	}
	buf := &bytes.Buffer{}
	t := template.Must(template.New("index").Parse(string(indexTpl)))
	err = t.Execute(buf, &tplObj)
	if err != nil {
		log.Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(buf.Bytes())
}
