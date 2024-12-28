package crudui

import (
	"database/sql"
)

// Controller is the main component that gets and saves objects in the database and generates HTTP handler
// that can be attached to an HTTP server.
type Controller struct {
	orm               ORM
	uriStructNameFunc map[string]map[string]func() interface{}
	tagName           string
	passFunc          func(string) string
	intFieldValues    map[string]IntFieldValues
	stringFieldValues map[string]StringFieldValues
}

func (c *Controller) GetORM() ORM {
	return c.orm
}

type ControllerConfig struct {
	TagName           string
	PasswordGenerator func(string) string
	IntFieldValues    map[string]IntFieldValues
	StringFieldValues map[string]StringFieldValues
	ORM               ORM
}

type ContextValue string

type IntFieldValues struct {
	Type   int
	Values map[int]string
}

type StringFieldValues struct {
	Type   int
	Values map[string]string
}

var ValuesMultipleBitChoice = 1
var ValuesSingleChoice = 2

// NewController returns new Controller object
func NewController(dbConn *sql.DB, tblPrefix string, cfg *ControllerConfig) *Controller {
	c := &Controller{}

	tagName := "ui"
	if cfg != nil && cfg.TagName != "" {
		tagName = cfg.TagName
	}
	c.tagName = tagName

	if cfg != nil && cfg.PasswordGenerator != nil {
		c.passFunc = cfg.PasswordGenerator
	}

	if cfg != nil {
		c.intFieldValues = cfg.IntFieldValues
		c.stringFieldValues = cfg.StringFieldValues
	}

	if cfg != nil && cfg.ORM != nil {
		c.orm = cfg.ORM
	} else {
		c.orm = newWrappedStruct2db(dbConn, tblPrefix, c.tagName)
	}

	return c
}
