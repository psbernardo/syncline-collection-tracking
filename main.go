package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/microsoft/go-mssqldb"
)

func main() {
	//connectionString := "sqlserver://localhost:1433?database=CTS_DEV&trusted_connection=true&encrypt=disable"
	connectionString := "sqlserver://dev:trustno1@localhost:1433?database=CTS_DEV"
	db, err := sql.Open("sqlserver", connectionString)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Error connecting to SQL Server:", err)
	}

	fmt.Println("Successfully connected to SQL Server!")
}
