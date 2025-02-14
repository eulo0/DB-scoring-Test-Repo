package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

var (
	userQueries = make(map[string]string[]) // maps username to queries
	sqlLoadOnce sync.Once
)

// at the moment i am not storing a query along with its expected value in memory since it may be expensive

// holds a query along with its expected answer
type QueryResult struct {
	Query    string                              `yaml:"query"`
	Expected []map[string]map[string]interface{} `yaml:"expected"`
}

func establishDBconnection(service Service, address string, user string, password string) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sql_timeout*time.Second)
	defer cancel()

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		user, password, address, service.Port, service.DBName)
	fmt.Printf("Attempting to connect with: %s\n", connStr)

	sqlConn, err := sqlx.Connect(
		"mysql",
		connStr,
	)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	err = sqlConn.PingContext(ctx)
	if err != nil {
		sqlConn.Close()
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("connection timeout after 5 seconds")
		}
		return nil, err
	}

	return sqlConn, nil
}

// loadSQLFiles is a utility function that loads a files or all the files in a directory which are named
// after a user which contains queries (read/write operations) for that specific user. it then maps the
// queries to a user.
func LoadSQLFiles(path string) {
	// Ensure it can only be run once
	ftpLoadOnce.Do(func() {
		// Check if path is a file or a directory
		info, statErr := os.Stat(path)
		if statErr != nil {
			err = fmt.Errorf("failed to stat %s: %v", path, statErr)
			return
		}

		if info.IsDir() {
			// Process directory
			entries, e := os.ReadDir(path)
			if e != nil {
				err = fmt.Errorf("failed to read directory %s: %v", path, e)
				return
			}

			// Read all files in the directory
			for _, entry := range entries {
				if entry.IsDir() {
					continue // Skip subdirectories
				}
				filePath := path + "/" + entry.Name()
				if readErr := readAndStoreFile(filePath, entry.Name()); readErr != nil {
					err = readErr
					return
				}
			}
		} else {
			// Process a single file
			err = readAndStoreFile(path, info.Name())
		}
	})

	return err
}

func ScoreSQLLogin(service Service) (int, bool, error) {
	conn, err := establishDBconnection(service, "localhost", "trump", "skibidigronk")
	if err != nil {
		fmt.Printf("Connection failed with error: %v\n", err)
		fmt.Printf("Trying to connect with:\nHost: localhost\nPort: %d\nUser: trump\nDB: %s\n",
			service.Port, service.DBName)
		return 0, false, err
	}
	defer conn.Close()
	return 1, true, nil
}

func ScoreSQLQuery(service Service, address string) (int, bool, error) {
	var user, pass string
	var err error
	// Check if the login user should be a single user
	if service.User != "" {
		user, pass = service.User, service.Password
	} else { // Otherwise read from the related query file
		user, pass, err = ChooseRandomUser(service.QFile)
		if err != nil {
			return 0, false, err
		}
	}
	conn, err := establishDBconnection(service, service.DBName, user, pass)
	if err != nil {
		fmt.Printf("Connection failed with error: %v\n", err)
		fmt.Printf("Trying to connect with:\nHost: localhost\nPort: %d\nUser: trump\nDB: %s\n",
			service.Port, service.DBName)
		return 0, false, err
	}
	defer conn.Close()
	_, err = compareResults(conn, service.QFile) // change this to the new struct we have in place
	if err != nil {
		return 0, false, err
	}
	return 1, true, nil
}

// # UTILITY FUNCTIONS # //

func compareResults(db *sqlx.DB, yamlFile string) (bool, error) {
	fmt.Println("Starting comparison...")

	content, err := os.ReadFile(yamlFile)
	if err != nil {
		return false, fmt.Errorf("failed to read YAML file: %v", err)
	}

	var testFile TestFile
	if err := yaml.Unmarshal(content, &testFile); err != nil {
		return false, fmt.Errorf("failed to parse YAML: %v", err)
	}

	fmt.Printf("Found %d queries to check\n", len(testFile.Queries))

	for i, queryTest := range testFile.Queries {
		fmt.Printf("\nExecuting query %d: %s\n", i+1, queryTest.Query)

		rows, err := db.Queryx(queryTest.Query)
		if err != nil {
			return false, fmt.Errorf("query failed: %v", err)
		}
		defer rows.Close()

		var actualResults []map[string]map[string]interface{}
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return false, fmt.Errorf("scan failed: %v", err)
			}

			typedRow := make(map[string]map[string]interface{})
			for k, v := range row {
				typedRow[k] = map[string]interface{}{
					"value": v,
					"type":  fmt.Sprintf("%T", v),
				}
			}
			actualResults = append(actualResults, typedRow)
		}

		fmt.Printf("Expected %d rows, got %d rows\n",
			len(queryTest.Expected), len(actualResults))

		// Check if number of rows match
		if len(queryTest.Expected) != len(actualResults) {
			fmt.Printf("Row count mismatch. Expected: %d, Got: %d\n",
				len(queryTest.Expected), len(actualResults))
			return false, nil
		}

		// Compare each row
		for rowIndex := range queryTest.Expected {
			if !compareValues(queryTest.Expected[rowIndex], actualResults[rowIndex]) {
				fmt.Printf("\nMismatch in row %d:\n", rowIndex)
				for field := range queryTest.Expected[rowIndex] {
					fmt.Printf("\nField: %s\n", field)
					fmt.Printf("Expected type: %v\n", queryTest.Expected[rowIndex][field]["type"])
					fmt.Printf("Got type: %v\n", actualResults[rowIndex][field]["type"])
					fmt.Printf("Expected value: %#v\n", queryTest.Expected[rowIndex][field]["value"])
					fmt.Printf("Got value: %#v\n", actualResults[rowIndex][field]["value"])
				}
				return false, nil
			}
		}

		fmt.Printf("Query %d passed successfully\n", i+1)
	}

	fmt.Println("\nAll queries matched expected results!")
	return true, nil
}

func compareValues(expected, actual map[string]map[string]interface{}) bool {
	if len(expected) != len(actual) {
		return false
	}

	for key, expectedField := range expected {
		actualField, exists := actual[key]
		if !exists {
			return false
		}

		expectedType := expectedField["type"].(string)
		actualType := actualField["type"].(string)
		if expectedType != actualType {
			return false
		}

		expectedValue := expectedField["value"]
		actualValue := actualField["value"]

		switch expectedType {
		case "int64":
			// Handle both int and int64 cases
			var expectedInt, actualInt int64

			switch ev := expectedValue.(type) {
			case int:
				expectedInt = int64(ev)
			case int64:
				expectedInt = ev
			}
			switch av := actualValue.(type) {
			case int:
				actualInt = int64(av)
			case int64:
				actualInt = av
			}

			if expectedInt != actualInt {
				return false
			}
		case "[]uint8":
			var expectedBytes []uint8
			if exp, ok := expectedValue.([]interface{}); ok {
				expectedBytes = make([]uint8, len(exp))
				for i, v := range exp {
					if num, ok := v.(int); ok {
						expectedBytes[i] = uint8(num)
					}
				}
			} else {
				expectedBytes = expectedValue.([]uint8)
			}

			actualBytes := actualValue.([]uint8)
			if len(expectedBytes) != len(actualBytes) {
				return false
			}
			for i := range expectedBytes {
				if expectedBytes[i] != actualBytes[i] {
					return false
				}
			}
		case "time.Time":
			if !expectedValue.(time.Time).Equal(actualValue.(time.Time)) {
				return false
			}
		default:
			if expectedValue != actualValue {
				return false
			}
		}
	}
	return true
}
