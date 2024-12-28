package crudui

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
)

// Global vars used across all the tests
var dbUser = "ui"
var dbPass = "ui123"
var dbName = "ui"
var dbConn *sql.DB

var dockerPool *dockertest.Pool
var dockerResource *dockertest.Resource

var httpPort = "32778"
var httpCancelCtx context.CancelFunc
var httpURI = "/ui/v1/"

var testController *Controller

// Test structs that should appear in the UI
type Person struct {
	ID       int64
	Flags    int64
	Name     string `ui:"req lenmin:5 lenmax:200"`
	Age      int    `ui:"req valmin:0 valmax:150"`
	PostCode string `ui_regexp:"^[0-9][0-9]-[0-9][0-9][0-9]$"`
	Email    string `ui:"req email"`
}

type Group struct {
	ID          int64
	Flags       int64
	Name        string `ui:"req lenmin:3 lenmax:100"`
	Description string `ui:"lenmax:5000"`
}

func TestMain(m *testing.M) {
	createDocker()
	defer removeDocker()

	createController()
	createDBStructure()
	createHTTPServer()

	code := m.Run()
	os.Exit(code)
}

func createDocker() {
	var err error
	dockerPool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	dockerResource, err = dockerPool.Run("postgres", "13", []string{"POSTGRES_PASSWORD=" + dbPass, "POSTGRES_USER=" + dbUser, "POSTGRES_DB=" + dbName})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	if err = dockerPool.Retry(func() error {
		var err error
		dbConn, err = sql.Open("postgres", fmt.Sprintf("host=localhost user=%s password=%s port=%s dbname=%s sslmode=disable", dbUser, dbPass, dockerResource.GetPort("5432/tcp"), dbName))
		if err != nil {
			return err
		}
		return dbConn.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
}

func createController() {
	testController = NewController(dbConn, "ui_", nil)
}

func createDBStructure() {
	testController.orm.CreateTables(&Person{})
	testController.orm.CreateTables(&Group{})
}

func createHTTPServer() {
	var ctx context.Context
	ctx, httpCancelCtx = context.WithCancel(context.Background())
	go func(ctx context.Context) {
		go func() {
			http.Handle(httpURI, testController.Handler(
				httpURI,
				func() interface{} { return &Person{} },
				func() interface{} { return &Group{} },
			))
			http.ListenAndServe(":"+httpPort, nil)
		}()
	}(ctx)
	time.Sleep(2 * time.Second)
}

func removeDocker() {
	dockerPool.Purge(dockerResource)
}
