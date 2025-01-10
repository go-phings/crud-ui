package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	crudui "github.com/go-phings/crud-ui"
	structdbpostgres "github.com/go-phings/struct-db-postgres"

	"github.com/go-phings/umbrella"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const dbDSN = "host=localhost user=cruduser password=crudpass port=54321 dbname=crud sslmode=disable"
const tblPrefix = "p_"

func main() {
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		log.Fatal("Error connecting to db")
	}

	orm := structdbpostgres.NewController(db, tblPrefix, &structdbpostgres.ControllerConfig{
		TagName: "crud",
	})
	err = orm.CreateTables(&User{}, &Item{}, &ItemGroup{}, &umbrella.Session{}, &umbrella.Permission{})
	if err != nil {
		log.Fatalf("Error creating table: %s", err.Error())
	}

	// add some dummy values to the database
	item := &Item{}
	itemGroup := &ItemGroup{}
	for i := 0; i < 301; i++ {
		item.ID = 0
		item.Flags = int64(i)
		item.Title = fmt.Sprintf("Item %d", i)
		item.Text = fmt.Sprintf("Description %d", i)
		orm.Save(item, structdbpostgres.SaveOptions{})
	}
	for i := 0; i < 73; i++ {
		itemGroup.ID = 0
		itemGroup.Flags = int64(i)
		itemGroup.Name = fmt.Sprintf("Name %d", i)
		itemGroup.Description = fmt.Sprintf("Description %d", i)
		orm.Save(itemGroup, structdbpostgres.SaveOptions{})
	}
	// end of dummy values

	uiCtl := crudui.NewController(db, tblPrefix, &crudui.ControllerConfig{
		TagName: "ui",
		PasswordGenerator: func(pass string) string {
			passEncrypted, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			if err != nil {
				return ""
			}
			return string(passEncrypted)
		},
		IntFieldValues: map[string]crudui.IntFieldValues{
			"Session_Flags": {
				Type:   crudui.ValuesSingleChoice,
				Values: umbrella.GetSessionFlagsSingleChoice(),
			},
			"Permission_Flags": {
				Type:   crudui.ValuesMultipleBitChoice,
				Values: umbrella.GetPermissionFlagsMultipleBitChoice(),
			},
			"Permission_ForType": {
				Type:   crudui.ValuesSingleChoice,
				Values: umbrella.GetPermissionForTypeSingleChoice(),
			},
			"Permission_Ops": {
				Type:   crudui.ValuesMultipleBitChoice,
				Values: umbrella.GetPermissionOpsMultipleBitChoice(),
			},
			"User_Flags": {
				Type:   crudui.ValuesMultipleBitChoice,
				Values: GetUserFlagsMultipleBitChoice(),
			},
		},
		StringFieldValues: map[string]crudui.StringFieldValues{
			"Permission_ToType": {
				Type: crudui.ValuesSingleChoice,
				Values: map[string]string{
					"all":        "all",
					"User":       "User",
					"Session":    "Session",
					"Permission": "Permission",
					"Item":       "Item",
					"ItemGroup":  "ItemGroup",
				},
			},
		},
	})

	http.Handle("/ui/",
		uiCtl.Handler(
			"/ui/",
			func() interface{} { return &User{} },
			func() interface{} { return &Item{} },
			func() interface{} { return &ItemGroup{} },
			func() interface{} { return &umbrella.Session{} },
			func() interface{} { return &umbrella.Permission{} },
		),
	)

	log.Fatal(http.ListenAndServe(":9001", nil))
}
