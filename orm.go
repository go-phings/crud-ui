package crudui

import (
	"database/sql"

	struct2db "github.com/go-phings/struct-db-postgres"
)

type ORMError interface {
	// IsInvalidFilters returns true when error is caused by invalid value of the filters when getting objects
	IsInvalidFilters() bool
	// Unwraps unwarps the original error
	Unwrap() error
	// Error returns error string
	Error() string
}

type ORM interface {
	// RegisterStruct initializes a specific object. ORMs often need to reflect the object to get the fields, build SQL queries etc.
	// When doing that, certain things such as tags can be inherited from another object. This is in the scenario where there is a root object (eg. Product) that contains all the validation tags and
	// another struct with less fields should be used as an input for API (eg. Product_WithoutCertainFields). In such case, there is no need to re-define tags such as validation.
	// Parameter `forceNameForDB` allows forcing another struct name (which later is used for generating table name).
	// This interface is based on the struct2db module and that module allows some cascade operations (such as delete or update). For this to work, and when certain fields are other structs, ORM must go
	// deeper and initializes that guys as well. When setting useOnlyRootFromInheritedObj to true, it's being avoided.
	RegisterStruct(obj interface{}, inheritFromObj interface{}, overwriteExisting bool, forceNameForDB string, useOnlyRootFromInheritedObj bool) ORMError
	// CreateTables create database tables for struct instances
	CreateTables(objs ...interface{}) error
	// DeleteMultiple removes struct items by their ids
	DeleteMultiple(obj interface{}, filters map[string]interface{}) error
	// Get fetches data from the database and returns struct instances. Hence, it requires a constructor for the returned objects. Apart from the self-explanatory fields, filters in a format of (field name, any value)
	// can be added, and each returned object (based on a database row) can be transformed into anything else.
	Get(newObjFunc func() interface{}, order []string, limit int, offset int, filters map[string]interface{}, rowObjTransformFunc func(interface{}) interface{}) ([]interface{}, ORMError)
	// GetCount returns number of struct items found in the database
	GetCount(newObjFunc func() interface{}, filters map[string]interface{}) (int64, ORMError)
	// Load populates struct instance's field values with database values
	Load(obj interface{}, id string) error
	// Save stores (creates or updates) struct instance in the appropriate database table
	Save(obj interface{}) error
	// Delete removes struct instance from the database table
	Delete(obj interface{}) error
}

// wrapped struct2db is an implementation of ORM interface that uses struct2db module
func newWrappedStruct2db(dbConn *sql.DB, tblPrefix string, tagName string) *wrappedStruct2db {
	c := &wrappedStruct2db{
		dbConn:    dbConn,
		tblPrefix: tblPrefix,
		tagName:   tagName,
	}
	c.orm = struct2db.NewController(dbConn, tblPrefix, &struct2db.ControllerConfig{
		TagName: c.tagName,
	})
	return c
}

type ormErrorImpl struct {
	op  string
	err error
}

func (o ormErrorImpl) IsInvalidFilters() bool {
	return o.op == "ValidateFilters"
}

func (o ormErrorImpl) Error() string {
	return o.err.Error()
}

func (o ormErrorImpl) Unwrap() error {
	return o.err
}

type wrappedStruct2db struct {
	dbConn    *sql.DB
	tblPrefix string
	tagName   string
	orm       *struct2db.Controller
}

func (w *wrappedStruct2db) DeleteMultiple(obj interface{}, filters map[string]interface{}) error {
	return w.orm.DeleteMultiple(obj, struct2db.DeleteMultipleOptions{
		Filters: filters,
	})
}

func (w *wrappedStruct2db) Get(newObjFunc func() interface{}, order []string, limit int, offset int, filters map[string]interface{}, rowObjTransformFunc func(interface{}) interface{}) ([]interface{}, ORMError) {
	xobj, err := w.orm.Get(newObjFunc, struct2db.GetOptions{
		Order:               order,
		Limit:               limit,
		Offset:              offset,
		Filters:             filters,
		RowObjTransformFunc: rowObjTransformFunc,
	})

	if err != nil {
		return nil, ormErrorImpl{
			op:  err.(struct2db.ErrController).Op,
			err: err.(struct2db.ErrController).Err,
		}
	}

	return xobj, nil
}

func (w *wrappedStruct2db) GetCount(newObjFunc func() interface{}, filters map[string]interface{}) (int64, ORMError) {
	cnt, err := w.orm.GetCount(newObjFunc, struct2db.GetCountOptions{
		Filters: filters,
	})

	if err != nil {
		return 0, ormErrorImpl{
			op:  err.(struct2db.ErrController).Op,
			err: err.(struct2db.ErrController).Err,
		}
	}

	return cnt, nil
}

func (w *wrappedStruct2db) CreateTables(objs ...interface{}) error {
	return w.orm.CreateTables(objs...)
}

func (w *wrappedStruct2db) Load(obj interface{}, id string) error {
	return w.orm.Load(obj, id, struct2db.LoadOptions{})
}

func (w *wrappedStruct2db) Save(obj interface{}) error {
	return w.orm.Save(obj, struct2db.SaveOptions{})
}

func (w *wrappedStruct2db) Delete(obj interface{}) error {
	return w.orm.Delete(obj, struct2db.DeleteOptions{})
}

func (w *wrappedStruct2db) RegisterStruct(obj interface{}, inheritFromObj interface{}, overwriteExisting bool, forceNameForDB string, useOnlyRootFromInheritedObj bool) ORMError {
	err := w.orm.AddSQLGenerator(obj, inheritFromObj, overwriteExisting, forceNameForDB, useOnlyRootFromInheritedObj)
	if err != nil {
		return ormErrorImpl{
			op:  err.(struct2db.ErrController).Op,
			err: err.(struct2db.ErrController).Err,
		}
	}
	return nil
}
