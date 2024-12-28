package crudui

import (
	"fmt"
	"html"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	validator "github.com/go-phings/struct-validator"
)

func (c *Controller) getStructItemFieldsHTML(u interface{}, values map[string]string) string {
	fieldHTMLs := validator.GenerateHTML(u, &validator.HTMLOptions{
		OverwriteTagName: c.tagName,
		ExcludeFields: map[string]bool{
			"ID": true,
		},
		OverwriteValues: values,
		FieldValues:     true,
	})

	htm := ""

	reName := regexp.MustCompile(`name="[A-Za-z0-9\-_]+"`)

	v := reflect.ValueOf(u)
	elem := v.Elem()
	i := reflect.Indirect(v)
	s := i.Type()
	structName := s.Name()

	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)

		if field.Name == "ID" {
			continue
		}

		gotDoubleEntry := c.isFieldHasTag(field, "dblentry")

		var value string
		fieldType := field.Type.Kind()
		if fieldType == reflect.String {
			value = elem.Field(j).String()
		}
		if fieldType == reflect.Bool {
			value = fmt.Sprintf("%v", elem.Field(j).Bool())
		}
		if c.isFieldInt(field) {
			value = fmt.Sprintf("%d", elem.Field(j).Int())
		}

		overwriteValue, ok := values[field.Name]
		if ok {
			value = overwriteValue
		}

		fieldHTML := c.getStructItemFieldHTML(field, structName, value, true)

		if fieldHTML == "" {
			fieldHTML = fieldHTMLs[field.Name]
		}

		htm += fmt.Sprintf("<p><label>%s</label>%s</p>", field.Name, fieldHTML)

		// TODO: very hacky
		if gotDoubleEntry {
			htmAgain := fmt.Sprintf("<p><label>%s (repeat)</label>%s</p>", field.Name, fieldHTMLs[field.Name])
			htm += reName.ReplaceAllString(htmAgain, fmt.Sprintf("name=\"%s___repeat\"", field.Name))
		}
	}

	return htm
}

func (c *Controller) isFieldHasTag(field reflect.StructField, tag string) bool {
	fieldTags := field.Tag.Get(c.tagName)
	if fieldTags != "" {
		fieldTags := strings.Split(fieldTags, " ")
		for _, ft := range fieldTags {
			if ft == tag {
				return true
			}
		}
	}
	return false
}

func (c *Controller) getStructItemFieldHTML(field reflect.StructField, structName string, value string, forEdit bool) string {
	h := ""
	if c.intFieldValues != nil && c.isFieldInt(field) {
		fv, ok := c.intFieldValues[structName+"_"+field.Name]
		if ok {
			if fv.Type == ValuesMultipleBitChoice {
				for ok, ov := range fv.Values {
					found := false
					if value != "" {
						i64, err := strconv.ParseInt(value, 10, 64)
						if err == nil {
							if i64&int64(ok) > 0 {
								found = true
							}
						}
					}
					if forEdit {
						checked := ""
						if found {
							checked = " checked"
						}
						h += fmt.Sprintf(`<input%s type="checkbox" name="%s" value="%d"/> %s`, checked, field.Name, ok, html.EscapeString(ov))
						continue
					}

					// not for edit
					if found {
						if h != "" {
							h += ", "
						}
						h += html.EscapeString(ov)
					}
				}
			}
			if fv.Type == ValuesSingleChoice {
				if forEdit {
					h += fmt.Sprintf(`<select name="%s">`, field.Name)
					for ok, ov := range fv.Values {
						selected := ""
						if value == fmt.Sprintf("%d", ok) {
							selected = " selected"
						}
						h += fmt.Sprintf(`<option%s value="%d">%s</option>`, selected, ok, html.EscapeString(ov))
					}
					h += "</select>"
				} else {
					for ok, ov := range fv.Values {
						if value == fmt.Sprintf("%d", ok) {
							h += html.EscapeString(ov)
						}
					}
				}

			}
		}
	}

	if h != "" {
		return h
	}

	if c.stringFieldValues != nil && field.Type.Kind() == reflect.String {
		fv, ok := c.stringFieldValues[structName+"_"+field.Name]
		if ok {
			if fv.Type == ValuesSingleChoice {
				if forEdit {
					h = fmt.Sprintf(`<select name="%s">`, field.Name)
					for ok, ov := range fv.Values {
						selected := ""
						if value == ok {
							selected = " selected"
						}
						h += fmt.Sprintf(`<option%s value="%s">%s</option>`, selected, html.EscapeString(ok), html.EscapeString(ov))
					}
					h += "</select>"
				} else {
					for ok, ov := range fv.Values {
						if value == ok {
							h += html.EscapeString(ov)
						}
					}
				}
			}
		}
	}

	return h
}

func (c *Controller) isFieldInt(field reflect.StructField) bool {
	k := field.Type.Kind()
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}
