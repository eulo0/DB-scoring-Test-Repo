package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"golang.org/x/exp/rand"
	"gopkg.in/yaml.v3"
)

var (
	userQueries = make(map[string]string) // maps username to filepath containing their queries
	sqlLoadOnce sync.Once
)

// holds a query along with its expected answer
type QueryResult struct {
	Query    string                              `yaml:"query"`
	Expected []map[string]map[string]interface{} `yaml:"expected"`
}

type QueryFile struct {
	Queries []QueryResult `yaml:"queries"`
}

func establishDBconnection(service Service, address string, user string, password string) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sql_timeout*time.Second)
	defer cancel()

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		user, password, address, service.Port, service.DBName)

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

// loadSQLFiles is a utility function that loads all the files in a directory which are named
// after a user which contains queries (read/write operations) for that specific user.

func LoadSQLFiles(path string) error {
	var err error

	sqlLoadOnce.Do(func() {
		// Check if path is a file or a directory
		info, err := os.Stat(path)
		if err != nil {
			err = fmt.Errorf("failed to stat %s: %v", path, err)
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

				filename := entry.Name()
				// Skip non-yaml files
				if !strings.HasSuffix(filename, ".yaml") {
					continue
				}

				// Get username by removing .yaml extension
				username := strings.TrimSuffix(filename, ".yaml")

				filePath := filepath.Join(path, filename)

				// Store in userQueries map
				userQueries[username] = filePath
				fmt.Printf("Stored query for user %s: %s\n", username, userQueries[username])
			}
		} else {
			// Process a single file
			filename := info.Name()
			if !strings.HasSuffix(filename, ".yaml") {
				err = fmt.Errorf("invalid query file: %s (must be .yaml)", filename)
				return
			}

			username := strings.TrimSuffix(filename, ".yaml")
			userQueries[username] = path
			fmt.Printf("Stored query for user %s: %s\n", username, userQueries[username])
		}
	})
	return err
}

func ScoreSQLLogin(service Service, address string) (int, bool, error) {
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
	conn, err := establishDBconnection(service, address, user, pass)
	if err != nil {
		fmt.Printf("Connection failed with error: %v\n", err)
		return 0, false, err
	}
	defer conn.Close()
	return 1, true, nil
}

func ScoreSQLQuery(service Service, address string) (int, bool, error) {
	var user, pass string
	var err error
	if service.User != "" {
		user, pass = service.User, service.Password
	} else {
		user, pass, err = ChooseRandomUser(service.QFile)
		if err != nil {
			return 0, false, err
		}
	}

	conn, err := establishDBconnection(service, address, user, pass)
	if err != nil {
		fmt.Printf("Connection failed with error: %v\n", err)
		return 0, false, err
	}
	defer conn.Close()

	// Get the random query
	file := userQueries[user]
	queryResult, err := getRandomQuery(file)
	if err != nil {
		return 0, false, fmt.Errorf("failed to get random query: %v", err)
	}

	// Compare results
	matched, err := compareResults(conn, queryResult)
	if err != nil {
		return 0, false, fmt.Errorf("comparison failed: %v", err)
	}
	if !matched {
		return 0, false, nil
	}

	return 1, true, nil
}

func getRandomQuery(filePath string) (*QueryResult, error) {
	// Read file content
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Parse YAML
	var queryFile QueryFile
	if err := yaml.Unmarshal(buf.Bytes(), &queryFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Check if there are any queries
	if len(queryFile.Queries) == 0 {
		return nil, fmt.Errorf("no queries found in file")
	}

	// Use golang.org/x/exp/rand with time seed
	src := rand.NewSource(uint64(time.Now().UnixNano()))
	rng := rand.New(src)

	// Select random query
	randomQueryIndex := rng.Intn(len(queryFile.Queries))
	selectedQuery := queryFile.Queries[randomQueryIndex]

	// Check if there are any expected results
	if len(selectedQuery.Expected) == 0 {
		return nil, fmt.Errorf("no expected results for query")
	}

	return &selectedQuery, nil
}

func randomQueryTest(service Service) error {
	user, _, err := ChooseRandomUser(service.QFile)
	if err != nil {
		return fmt.Errorf("failed to get random user: %v", err)
	}

	file, exists := userQueries[user]
	if !exists {
		return fmt.Errorf("no query file found for user %s", user)
	}

	queryResult, err := getRandomQuery(file)
	if err != nil {
		return fmt.Errorf("failed to get random query and result: %v", err)
	}

	fmt.Printf("Selected User: %s\n", user)
	fmt.Printf("Random Query: %s\n", queryResult.Query)
	fmt.Printf("\nExpected Results:\n")
	for i, result := range queryResult.Expected {
		fmt.Printf("Result %d:\n", i+1)
		for field, value := range result {
			fmt.Printf("  %s: (Type: %s, Value: %v)\n",
				field, value["type"], value["value"])
		}
		fmt.Println()
	}

	return nil
}

func compareResults(db *sqlx.DB, queryResult *QueryResult) (bool, error) {
	rows, err := db.Queryx(queryResult.Query)
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

	if len(actualResults) == 0 {
		return false, fmt.Errorf("no results returned from query")
	}

	// First, ensure we have the same number of rows
	if len(actualResults) != len(queryResult.Expected) {
		fmt.Printf("Row count mismatch: expected %d rows, got %d rows\n",
			len(queryResult.Expected), len(actualResults))
		return false, nil
	}

	// Compare all rows
	for i, expectedRow := range queryResult.Expected {
		matchFound := false
		for _, actualRow := range actualResults {
			if compareValues(expectedRow, actualRow) {
				matchFound = true
				break
			}
		}
		if !matchFound {
			fmt.Printf("No match found for expected row %d\n", i)
			return false, nil
		}
	}

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

		expectedType, ok := expectedField["type"].(string)
		if !ok {
			return false
		}

		actualType, ok := actualField["type"].(string)
		if !ok {
			return false
		}

		if expectedType != actualType {
			return false
		}

		expectedValue := expectedField["value"]
		actualValue := actualField["value"]

		switch expectedType {
		case "int64":
			var expectedInt, actualInt int64

			switch ev := expectedValue.(type) {
			case int:
				expectedInt = int64(ev)
			case int64:
				expectedInt = ev
			default:
				return false
			}

			switch av := actualValue.(type) {
			case int:
				actualInt = int64(av)
			case int64:
				actualInt = av
			default:
				return false
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
					} else {
						return false
					}
				}
			} else if bytes, ok := expectedValue.([]uint8); ok {
				expectedBytes = bytes
			} else {
				return false
			}

			actualBytes, ok := actualValue.([]uint8)
			if !ok {
				return false
			}

			if len(expectedBytes) != len(actualBytes) {
				return false
			}

			for i := range expectedBytes {
				if expectedBytes[i] != actualBytes[i] {
					return false
				}
			}

		case "time.Time":
			et, ok1 := expectedValue.(time.Time)
			at, ok2 := actualValue.(time.Time)
			if !ok1 || !ok2 {
				return false
			}
			if !et.Equal(at) {
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
