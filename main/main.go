package main

import (
	"fmt"
	"log"
	"path/filepath"
)

func trivial(boolean bool) {
	if boolean {
		fmt.Println("Success")
	} else {
		fmt.Println("Failed")
	}
}

func main() {
	dir, err := filepath.Abs(".")
	if err != nil {
		log.Fatal(err)
	}

	qdirpath := filepath.Join(dir, "../service_queries/sqlfiles")

	DB := Service{
		QFile:  filepath.Join(qdirpath, "users.txt"),
		QDir:   qdirpath,
		Port:   5010,
		DBName: "minecraft",
	}

	Initalize()
	err = LoadSQLFiles(DB.QDir)

	fmt.Println("\nRunning Login Test:")
	_, boolean, _ := ScoreSQLLogin(DB, "localhost")
	trivial(boolean)

	err = LoadSQLFiles(DB.QDir)
	if err != nil {
		fmt.Println("Error Loading SQL Files")
	}

	fmt.Println("\nRunning Query Test:")
	_, boolean, err = ScoreSQLQuery(DB, "localhost")
	trivial(boolean)
}
