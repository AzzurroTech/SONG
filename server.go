package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	veni "github.com/Emperor42/veni"
	vidi "github.com/Emperor42/vidi"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Opening a driver typically will not attempt to connect to the database.
	db, err := sql.Open("mysql", "user:password@/dbname")
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	fmt.Println("DB has been setup")
	//root server handling
	http.Handle("/", veni.Load(http.Dir("./root")))
	//Other server handling
	http.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir("./data"))))
	//Access Vici platform
	http.Handle("/demo/", http.StripPrefix("/demo/", http.FileServer(http.Dir("./vici"))))
	//Access Vidi database
	http.Handle("/base/", vidi.Load(db))
	fmt.Println("Listen on port 3000")
	//serve the server
	http.ListenAndServe(":3000", nil)
}
