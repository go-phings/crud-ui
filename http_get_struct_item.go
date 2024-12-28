package crudui

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"text/template"

	validator "github.com/go-phings/struct-validator"

	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type structItemTplObj struct {
	Name       string
	URI        string
	FieldsHTML string
	MsgHTML    string
	OnlyMsg    bool
	ID         string
	ReadOnly   bool
}

func (c *Controller) tryGetStructItem(w http.ResponseWriter, r *http.Request, uri string) bool {
	structName, id := c.getStructAndIDFromURI("x/struct_item/", c.getRealURI(uri, r.RequestURI))

	if structName == "" {
		return false
	}

	// Check if struct exists
	_, ok := c.uriStructNameFunc[uri][structName]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return true
	}

	// check access
	readOnly := false
	if id != "" {
		if !c.isStructOperationAllowed(r, structName, OpsRead) {
			w.WriteHeader(http.StatusForbidden)
			return true
		}
		if !c.isStructOperationAllowed(r, structName, OpsUpdate) {
			readOnly = true
		}
	} else {
		if !c.isStructOperationAllowed(r, structName, OpsCreate) {
			w.WriteHeader(http.StatusForbidden)
			return true
		}
	}

	// Render the page
	c.renderStructItem(w, r, uri, c.uriStructNameFunc[uri][structName], id, map[string]string{}, 0, "", readOnly)

	return true
}

func (c *Controller) tryStructItem(w http.ResponseWriter, r *http.Request, uri string) bool {
	structName, id := c.getStructAndIDFromURI("x/struct_item/", c.getRealURI(uri, r.RequestURI))

	if structName == "" {
		return false
	}

	// Check if struct exists
	_, ok := c.uriStructNameFunc[uri][structName]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return true
	}

	if r.Method != http.MethodPut && r.Method != http.MethodPost && r.Method != http.MethodDelete {
		return false
	}

	if r.Method == http.MethodDelete && id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return true
	}

	// check access for delete
	if r.Method == http.MethodDelete {
		if !c.isStructOperationAllowed(r, structName, OpsDelete) {
			w.WriteHeader(http.StatusForbidden)
			return true
		}
	}

	// check access for either create or update
	if id != "" {
		if !c.isStructOperationAllowed(r, structName, OpsUpdate) {
			w.WriteHeader(http.StatusForbidden)
			return true
		}
	} else {
		if !c.isStructOperationAllowed(r, structName, OpsCreate) {
			w.WriteHeader(http.StatusForbidden)
			return true
		}
	}

	obj := c.uriStructNameFunc[uri][structName]()
	// Set ID if present in the URI
	if id != "" {
		val := reflect.ValueOf(obj).Elem()
		valField := val.FieldByName("ID")
		if !valField.CanSet() {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		i, _ := strconv.ParseInt(id, 10, 64)
		valField.SetInt(i)

		// Load values because we might not overwrite all of them (eg. passwords might stay untouched)
		err := c.orm.Load(obj, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
	}

	// Handle delete here
	if r.Method == http.MethodDelete {
		err2 := c.orm.Delete(obj)
		if err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}

		c.renderMsg(w, r, MsgSuccess, fmt.Sprintf("%s item has been successfully deleted.", structName))
		return true
	}

	// Get form data
	r.ParseForm()

	// Create object, set value and validate it
	invalidFormFields := map[string]bool{}

	// Value for each form key is actually an array of strings. We're taking the first one here only
	// TODO: Tweak it
	v := reflect.ValueOf(obj)
	s := v.Elem()
	indir := reflect.Indirect(v)
	typ := indir.Type()

	postValues := map[string]string{}
	for fk, fv := range r.Form {
		postValues[fk] = fv[0]

		if fv[0] == "" {
			continue
		}

		f := s.FieldByName(fk)
		if f.IsValid() && f.CanSet() {
			// We can set password fields only when they are not empty

			field, _ := typ.FieldByName(fk)
			gotPassField := c.isFieldHasTag(field, "password")
			if gotPassField {
				if fv[0] == "" {
					continue
				}
				if c.passFunc != nil {
					passVal := c.passFunc(fv[0])
					if passVal == "" {
						w.WriteHeader(http.StatusInternalServerError)
						return true
					}
					f.SetString(passVal)
					continue
				}
			}

			if f.Kind() == reflect.String {
				f.SetString(fv[0])
			}

			if c.isFieldInt(field) {
				var iSum int64
				for _, fvv := range fv {
					i, err := strconv.ParseInt(fvv, 10, 64)
					if err != nil {
						invalidFormFields[fk] = true
						continue
					}
					if iSum&i == 0 {
						iSum += i
					}
				}

				if !invalidFormFields[fk] {
					f.SetInt(iSum)
				}
			}
		}
	}

	valid, failedFields := validator.Validate(obj, &validator.ValidationOptions{
		OverwriteTagName: "ui",
	})

	if len(invalidFormFields) > 0 {
		for k := range invalidFormFields {
			failedFields[k] = failedFields[k] | validator.FailRegexp
		}
	}

	// TODO: quick hack - if any '___repeat' exist then it should have the same value as field without it
	for fk, fv := range postValues {
		if strings.HasSuffix(fk, "___repeat") && fv != postValues[strings.Replace(fk, "___repeat", "", 1)] {
			valid = false
			failedFields[fk] = validator.Required
		}
	}

	if !valid || len(failedFields) > 0 {
		invVals := []string{}
		for k := range failedFields {
			invVals = append(invVals, k)
		}
		c.renderStructItem(w, r, uri, c.uriStructNameFunc[uri][structName], id, postValues, MsgFailure, fmt.Sprintf("The following fields have invalid values: %s", strings.Join(invVals, ",")), false)
		return true
	}

	err2 := c.orm.Save(obj)
	if err2 != nil {
		c.renderStructItem(w, r, uri, c.uriStructNameFunc[uri][structName], id, postValues, MsgFailure, fmt.Sprintf("Problem with saving: %s", err2.Error()), false)
		return true
	}

	// Update
	if id != "" {
		c.renderStructItem(w, r, uri, c.uriStructNameFunc[uri][structName], id, postValues, MsgSuccess, fmt.Sprintf("%s item has been successfully updated.", structName), false)
		return true
	}

	// Create
	c.renderMsg(w, r, MsgSuccess, fmt.Sprintf("%s item has been successfully added.", structName))
	return true
}

func (c *Controller) renderStructItem(w http.ResponseWriter, r *http.Request, uri string, objFunc func() interface{}, id string, postValues map[string]string, msgType int, msg string, readOnly bool) {
	tpl, err := c.getStructItemHTML(uri, objFunc, id, postValues, msgType, msg, readOnly)
	if err != nil {
		log.Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
		return
	}
	w.Write([]byte(tpl))
}

func (c *Controller) getStructItemHTML(uri string, objFunc func() interface{}, id string, postValues map[string]string, msgType int, msg string, readOnly bool) (string, error) {
	structItemTpl, err := embed.FS.ReadFile(htmlDir, "html/struct_item.html")
	if err != nil {
		return "", fmt.Errorf("error reading struct item template from embed: %w", err)
	}

	tplObj, err := c.getStructItemTplObj(uri, objFunc, id, postValues, msgType, msg, readOnly)
	if err != nil {
		return "", fmt.Errorf("error getting struct item for html: %w", err)
	}

	buf := &bytes.Buffer{}
	t := template.Must(template.New("structItem").Parse(string(structItemTpl)))
	err = t.Execute(buf, &tplObj)
	if err != nil {
		return "", fmt.Errorf("error processing struct item template: %w", err)
	}

	return buf.String(), nil
}

func (c *Controller) getStructItemTplObj(uri string, objFunc func() interface{}, id string, postValues map[string]string, msgType int, msg string, readOnly bool) (*structItemTplObj, error) {
	o := objFunc()

	if id != "" {
		err := c.orm.Load(o, id)
		if err != nil {
			return nil, err
		}
	}

	onlyMsg := false
	if msgType == MsgSuccess && id == "" {
		onlyMsg = true
	}

	a := &structItemTplObj{
		URI:        uri,
		Name:       getStructName(o),
		FieldsHTML: c.getStructItemFieldsHTML(o, postValues),
		MsgHTML:    c.getMsgHTML(msgType, msg),
		OnlyMsg:    onlyMsg,
		ID:         id,
		ReadOnly:   readOnly,
	}

	return a, nil
}
