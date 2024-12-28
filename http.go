package crudui

import (
	"embed"
	"fmt"
	"html"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

//go:embed html/*
var htmlDir embed.FS

const MsgSuccess = 1
const MsgFailure = 2

type HandlerConfig struct {
	UserNameFunc func(int) string
}

// Handler returns an HTTP handler that can be attached to HTTP server. It runs a simple UI that allows
// managing the data.
// Each of the func() argument should be funcs that create objects that are meant to be managed in the UI.
func (c *Controller) Handler(uri string, objFuncs ...func() interface{}) http.Handler {
	c.setStructNameFunc(uri, objFuncs...)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if c.tryGetLogin(w, r, uri) {
				return
			}

			if c.tryGetHome(w, r, uri, objFuncs...) {
				return
			}
			if c.tryGetStructList(w, r, uri, objFuncs...) {
				return
			}
			if c.tryGetStructItems(w, r, uri) {
				return
			}
			if c.tryGetStructItem(w, r, uri) {
				return
			}
		}

		if c.tryStructItem(w, r, uri) {
			return
		}
		if c.tryStructItems(w, r, uri) {
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	})
}

func (c *Controller) getStructAndIDFromURI(prefix string, uri string) (string, string) {
	if prefix != "" {
		uri = strings.Replace(uri, prefix, "", 1)
	}

	structName := ""
	id := ""

	re := regexp.MustCompile(`^([a-zA-Z0-9_]+)/([0-9]+)$`)
	re2 := regexp.MustCompile(`^([a-zA-Z0-9_]+)/$`)

	// Try matching to a URI with ID (edit item)
	matched := re.FindStringSubmatch(uri)
	if len(matched) == 3 {
		structName = matched[1]
		id = matched[2]
	}

	// Try matching to a URI without ID (add item)
	if structName == "" {
		matched2 := re2.FindStringSubmatch(uri)
		if len(matched2) == 2 {
			structName = matched2[1]
		}
	}

	return structName, id
}

func (c *Controller) getMsgHTML(msgType int, msg string) string {
	if msgType == 0 {
		return ""
	}
	t := "success"
	if msgType == MsgFailure {
		t = "error"
	}
	return fmt.Sprintf(`<div class="msg %s">%s</div>`, t, html.EscapeString(msg))
}

func (c *Controller) getRealURI(handlerURI string, requestURI string) string {
	uri := requestURI[len(handlerURI):]
	xs := strings.SplitN(uri, "?", 2)
	return xs[0]
}

func (c *Controller) setStructNameFunc(uri string, objFuncs ...func() interface{}) {
	// Loop through objFuncs and get their struct names
	if c.uriStructNameFunc == nil {
		c.uriStructNameFunc = make(map[string]map[string]func() interface{})
	}
	c.uriStructNameFunc[uri] = map[string]func() interface{}{}
	for _, obj := range objFuncs {
		o := obj()
		v := reflect.ValueOf(o)
		i := reflect.Indirect(v)
		s := i.Type()
		c.uriStructNameFunc[uri][s.Name()] = obj
	}
}

func (c *Controller) renderMsg(w http.ResponseWriter, r *http.Request, msgType int, msg string) {
	w.Write([]byte(c.getMsgHTML(msgType, msg)))
}
