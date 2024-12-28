package crudui

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"text/template"
)

type structListTplObj struct {
	Structs []*structListTplObjItem
	URI     string
}

type structListTplObjItem struct {
	Name string
}

func (c *Controller) tryGetStructList(w http.ResponseWriter, r *http.Request, uri string, objFuncs ...func() interface{}) bool {
	realURI := c.getRealURI(uri, r.RequestURI)
	if realURI == "x/struct_list/" {
		c.renderStructList(w, r, uri, objFuncs...)
		return true
	}
	return false
}

func (c *Controller) renderStructList(w http.ResponseWriter, r *http.Request, uri string, objFuncs ...func() interface{}) {
	tpl, err := c.getStructListHTML(uri, r, objFuncs...)
	if err != nil {
		log.Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
		return
	}
	w.Write([]byte(tpl))
}

func (c *Controller) getStructListHTML(uri string, r *http.Request, objFuncs ...func() interface{}) (string, error) {
	structListTpl, err := embed.FS.ReadFile(htmlDir, "html/struct_list.html")
	if err != nil {
		return "", fmt.Errorf("error reading struct list template from embed: %w", err)
	}

	tplObj, err := c.getStructListTplObj(uri, r, objFuncs...)
	if err != nil {
		return "", fmt.Errorf("error getting struct list for html: %w", err)
	}

	buf := &bytes.Buffer{}
	t := template.Must(template.New("structList").Parse(string(structListTpl)))
	err = t.Execute(buf, &tplObj)
	if err != nil {
		return "", fmt.Errorf("error processing struct list template: %w", err)
	}

	return buf.String(), nil
}

func (c *Controller) getStructListTplObj(uri string, r *http.Request, objFuncs ...func() interface{}) (*structListTplObj, error) {
	l := &structListTplObj{
		URI:     uri,
		Structs: []*structListTplObjItem{},
	}

	allowedTypes := r.Context().Value(ContextValue(fmt.Sprintf("AllowedTypes_%d", OpsList)))

	for _, objFunc := range objFuncs {
		o := objFunc()
		v := reflect.ValueOf(o)
		i := reflect.Indirect(v)
		s := i.Type()

		if allowedTypes != nil {
			v, ok := allowedTypes.(map[string]bool)[s.Name()]
			if !ok || !v {
				v2, ok2 := allowedTypes.(map[string]bool)["all"]
				if !ok2 || !v2 {
					continue
				}
			}
		}

		l.Structs = append(l.Structs, &structListTplObjItem{
			Name: s.Name(),
		})
	}

	return l, nil
}
