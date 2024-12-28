package crudui

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"log"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type structItemsTplObj struct {
	Name                  string
	URI                   string
	Fields                []string
	ItemsHTML             []interface{}
	ItemsCount            int64
	PageNumbers           []string
	ParamPage             string
	ParamLimit            string
	ParamRawFilterEscaped string
	ParamFiltersEscaped   map[string]string
	ParamOrder            string
	ParamOrderDirection   string
	CanCreate             bool
	CanUpdate             bool
	CanDelete             bool
}

type structItemsParams struct {
	Page           int
	Limit          int
	RawFilter      string
	FiltersForDB   map[string]interface{}
	FiltersForUI   map[string]string
	Order          string
	OrderDirection string
}

func (c *Controller) tryGetStructItems(w http.ResponseWriter, r *http.Request, uri string) bool {
	realURI := c.getRealURI(uri, r.RequestURI)
	if !strings.HasPrefix(realURI, "x/struct_items/") {
		return false
	}

	structName := strings.Replace(realURI, "x/struct_items/", "", 1)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_]+/$`, structName)
	if err != nil || !matched {
		w.WriteHeader(http.StatusBadRequest)
		return true
	}
	structName = strings.Replace(structName, "/", "", 1)

	_, ok := c.uriStructNameFunc[uri][structName]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return true
	}

	canCreate := true
	canUpdate := true
	canDelete := true
	// check access
	if !c.isStructOperationAllowed(r, structName, OpsRead) {
		w.WriteHeader(http.StatusForbidden)
		return true
	}
	if !c.isStructOperationAllowed(r, structName, OpsCreate) {
		canCreate = false
	}
	if !c.isStructOperationAllowed(r, structName, OpsUpdate) {
		canUpdate = false
	}
	if !c.isStructOperationAllowed(r, structName, OpsDelete) {
		canDelete = false
	}

	page := r.FormValue("page")
	limit := r.FormValue("limit")
	rawFilter := r.FormValue("rawFilter")
	order := r.FormValue("order")
	orderDirection := strings.ToLower(r.FormValue("orderDirection"))

	reNumber := regexp.MustCompile(`^[0-9]+$`)
	if !reNumber.MatchString(page) {
		page = "1"
	}
	if !reNumber.MatchString(limit) {
		limit = "25"
	}
	// todo: rawFilter
	pageInt, _ := strconv.ParseInt(page, 10, 64)
	limitInt, _ := strconv.ParseInt(limit, 10, 64)

	if orderDirection != "desc" {
		orderDirection = "asc"
	}

	obj := c.uriStructNameFunc[uri][structName]()

	if !isStructField(obj, order) {
		order = "ID"
	}

	filtersForDB := map[string]interface{}{}
	filtersForUI := map[string]string{}
	s := reflect.ValueOf(obj).Elem()
	for fk, fv := range r.Form {
		// TODO: fv is actually an array and we just take the first value
		if !strings.HasPrefix(fk, "filter") || strings.TrimSpace(fv[0]) == "" {
			continue
		}
		filterName := fk[6:]
		f := s.FieldByName(filterName)
		if !f.IsValid() {
			continue
		}
		filtersForDB[filterName+":%"] = "%" + fv[0] + "%"
		filtersForUI[filterName] = html.EscapeString(fv[0])
	}

	c.renderStructItems(w, r, uri, c.uriStructNameFunc[uri][structName], structItemsParams{
		Page:           int(pageInt),
		Limit:          int(limitInt),
		RawFilter:      rawFilter,
		Order:          order,
		OrderDirection: orderDirection,
		FiltersForDB:   filtersForDB,
		FiltersForUI:   filtersForUI,
	}, canCreate, canUpdate, canDelete)
	return true
}

func (c *Controller) tryStructItems(w http.ResponseWriter, r *http.Request, uri string) bool {
	structName, id := c.getStructAndIDFromURI("x/struct_items/", c.getRealURI(uri, r.RequestURI))

	if structName == "" || id != "" {
		return false
	}

	// Check if struct exists
	newObjFunc, ok := c.uriStructNameFunc[uri][structName]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return true
	}

	if r.Method != http.MethodPut && r.Method != http.MethodPost && r.Method != http.MethodDelete {
		return false
	}

	_ = c.uriStructNameFunc[uri][structName]()

	// Handle delete here
	if r.Method == http.MethodDelete {
		// Read ids from the param 'ids'
		ids := r.URL.Query().Get("ids")
		match, err := regexp.MatchString(`^[0-9]+(,[0-9]+)*$`, ids)
		if err != nil || !match {
			w.WriteHeader(http.StatusBadRequest)
			return true
		}

		// Run delete for multiple rows
		idsList := strings.Split(ids, ",")
		idsInt := []int64{}
		for _, id := range idsList {
			idInt, _ := strconv.ParseInt(id, 10, 64)
			idsInt = append(idsInt, idInt)
		}

		err2 := c.orm.DeleteMultiple(newObjFunc(), map[string]interface{}{
			"_raw": []interface{}{
				".ID IN (?)",
				idsInt,
			},
		})

		if err2 != nil {
			c.renderMsg(w, r, MsgFailure, fmt.Sprintf("Problem with removing %s items.", structName))
			return true
		}

		c.renderMsg(w, r, MsgSuccess, fmt.Sprintf("%s items have been successfully deleted.", structName))
		return true
	}

	// Any other request is invalid
	w.WriteHeader(http.StatusBadRequest)
	return true
}

func (c *Controller) renderStructItems(w http.ResponseWriter, r *http.Request, uri string, objFunc func() interface{}, params structItemsParams, canCreate bool, canUpdate bool, canDelete bool) {
	tpl, err := c.getStructItemsHTML(uri, objFunc, params, canCreate, canUpdate, canDelete)
	if err != nil {
		log.Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
		return
	}
	w.Write([]byte(tpl))
}

func (c *Controller) getStructItemsHTML(uri string, objFunc func() interface{}, params structItemsParams, canCreate bool, canUpdate bool, canDelete bool) (string, error) {
	structItemsTpl, err := embed.FS.ReadFile(htmlDir, "html/struct_items.html")
	if err != nil {
		return "", fmt.Errorf("error reading struct items template from embed: %w", err)
	}

	tplObj, err := c.getStructItemsTplObj(uri, objFunc, params, canCreate, canUpdate, canDelete)
	if err != nil {
		return "", fmt.Errorf("error getting struct items for html: %w", err)
	}

	buf := &bytes.Buffer{}
	t := template.Must(template.New("structItems").Funcs(template.FuncMap{
		"SplitRow": func(s string) []string {
			sArr := strings.SplitN(s, ":", 2)
			return sArr
		},
	}).Parse(string(structItemsTpl)))
	err = t.Execute(buf, &tplObj)
	if err != nil {
		return "", fmt.Errorf("error processing struct items template: %w", err)
	}
	return buf.String(), nil
}

func (c *Controller) getStructItemsTplObj(uri string, objFunc func() interface{}, params structItemsParams, canCreate bool, canUpdate bool, canDelete bool) (*structItemsTplObj, error) {
	o := objFunc()

	getPage, getLimit, getOffset := c.getPageLimitOffset(params.Page, params.Limit)

	itemsHTML, err := c.orm.Get(objFunc, []string{params.Order, params.OrderDirection}, getLimit, getOffset, params.FiltersForDB, func(obj interface{}) interface{} {
		out := ""
		id := ""

		v := reflect.ValueOf(obj)
		elem := v.Elem()
		i := reflect.Indirect(v)
		s := i.Type()
		structName := s.Name()
		for j := 0; j < s.NumField(); j++ {
			out += "<td>"
			field := s.Field(j)
			fieldType := field.Type.Kind()
			hideValue := c.isFieldHasTag(field, "hidden")
			if hideValue {
				out += "(hidden)</td>"
				continue
			}

			var value string
			if fieldType == reflect.String {
				value = elem.Field(j).String()
			}
			if fieldType == reflect.Bool {
				value = fmt.Sprintf("%v", elem.Field(j).Bool())
			}
			if fieldType == reflect.Int || fieldType == reflect.Int64 {
				value = fmt.Sprintf("%d", elem.Field(j).Int())
			}

			valueHTML := c.getStructItemFieldHTML(field, structName, value, false)
			if valueHTML != "" {
				out += valueHTML + "</td>"
				continue
			}

			out += html.EscapeString(value)
			out += "</td>"

			if field.Name == "ID" {
				id = fmt.Sprintf("%d", elem.Field(j).Int())
			}
		}

		return fmt.Sprintf("%s:%s", id, out)
	})
	if err != nil {
		return nil, err
	}

	itemsCount, err := c.orm.GetCount(objFunc, params.FiltersForDB)
	if err != nil {
		return nil, err
	}
	pageNumbers := c.getPageNumbers(itemsCount, getLimit, getPage)

	its := &structItemsTplObj{
		URI:                 uri,
		Name:                getStructName(o),
		Fields:              getStructFieldNames(o),
		ItemsHTML:           itemsHTML,
		ItemsCount:          itemsCount,
		ParamPage:           fmt.Sprintf("%d", getPage),
		ParamLimit:          fmt.Sprintf("%d", getLimit),
		PageNumbers:         pageNumbers,
		ParamOrder:          params.Order,
		ParamOrderDirection: params.OrderDirection,
		ParamFiltersEscaped: params.FiltersForUI,
		CanCreate:           canCreate,
		CanUpdate:           canUpdate,
		CanDelete:           canDelete,
	}

	return its, nil
}

func (c *Controller) getPageLimitOffset(page int, limit int) (int, int, int) {
	if limit < 1 {
		limit = 25
	}
	if page < 1 {
		page = 1
	}
	return page, limit, limit * (page - 1)
}

func (c *Controller) getPageNumbers(itemsCount int64, getLimit int, getPage int) []string {
	p := int(math.Ceil(float64(itemsCount) / float64(getLimit)))

	a := []string{}
	if p < 11 {
		for i := 1; i <= p; i++ {
			a = append(a, fmt.Sprintf("%d", i))
		}
		return a
	}
	a = append(a, "1")
	if getPage-2 > 1 {
		a = append(a, "")
	}
	if getPage-1 > 1 {
		a = append(a, fmt.Sprintf("%d", getPage-1))
	}
	if getPage > 1 && getPage < p {
		a = append(a, fmt.Sprintf("%d", getPage))
	}
	if getPage+1 < p {
		a = append(a, fmt.Sprintf("%d", getPage+1))
	}
	if getPage+2 < p {
		a = append(a, "")
	}
	a = append(a, fmt.Sprintf("%d", p))
	return a
}
