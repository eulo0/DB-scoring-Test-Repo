package main

import (
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

/*
func main() {

	DB := Service{
		Port:   5010,
		DBName: "minecraft",
	}
	Initalize()
	/*
		_, isSuccess, _ := ScoreSQLLogin(DB)
		fmt.Println(isSuccess)
		/*
			user, password, _ := ChooseRandomUser("C://Users//bobby//OneDrive//Desktop//NEST//lol//Q_DIR//users.txt")
			the_string := fmt.Sprintf("%s:%s", user, password)
			fmt.Println(the_string)
				if err := run(); err != nil {
					log.Fatal(err)
				}

	conn, err := establishDBconnection(DB, "localhost", "trump", "skibidigronk")
	if err != nil {
		generateTestFile(conn,
			"C://Users//bobby//OneDrive//Desktop//NEST//lol//main//queries.txt",
			"C://Users//bobby//OneDrive//Desktop//NEST//lol//main//queries_updated.yaml",
		)
	}
}
*/

/*
func main() {
	DB := Service{
		Port:   5010,
		DBName: "minecraft",
	}
	Initalize()

	conn, err := establishDBconnection(DB, "localhost", "trump", "skibidigronk")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	err = generateTestFile(conn,
		"C://Users//bobby//OneDrive//Desktop//NEST//lol//main//queries.txt",
		"C://Users//bobby//OneDrive//Desktop//NEST//lol//main//queries_updated.yaml",
	)
	if err != nil {
		log.Fatalf("Failed to generate test file: %v", err)
	}
}
*/

func main() {
	DB := Service{
		Port:   5010,
		DBName: "minecraft",
	}
	Initalize()

	conn, err := establishDBconnection(DB, "localhost", "trump", "skibidigronk")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	db, err := sqlx.Connect("mysql", "trump:skibidigronk@tcp(localhost:5010)/minecraft?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	matches, err := compareResults(db, "C://Users//bobby//OneDrive//Desktop//NEST//lol//service_queries//mysqlfiles//queriesold.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if matches {
		fmt.Println("All queries matched expected results!")
	} else {
		fmt.Println("Queries did not match expected results.")
	}
}
