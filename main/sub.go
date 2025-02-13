package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

var (
	sqlUsers    = make(map[string][]string)            // store users and their passwords here
	sqlQueries  = make(map[string]map[string][]string) // maps username to queries (read/write operations)
	sqlLoadOnce sync.Once
)

func establishDBconnection(service Service, address string, user string, password string) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Changed to 5 seconds for faster feedback
	defer cancel()

	// Print connection details
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

func ScoreSQLLogin(service Service) (int, bool, error) {
	conn, err := establishDBconnection(service, "localhost", "trump", "skibidigronk")
	if err != nil {
		fmt.Printf("Connection failed with error: %v\n", err)
		// Also print connection details for debugging
		fmt.Printf("Trying to connect with:\nHost: localhost\nPort: %d\nUser: trump\nDB: %s\n",
			service.Port, service.DBName)
		return 0, false, err
	}
	defer conn.Close()
	return 1, true, nil
}

/*
func LoadSQLFiles(path string) (error){

}

func ScoreSQLQuery(service enum.Service, address string) (int, bool, error) {
}

this will be released at a later time
func ScoreSQLSchema(service enum.Service, address string) (int, bool, error) {
}
*/

func deepCompareResults(expected, got []map[string]map[string]interface{}) bool {
	if len(expected) != len(got) {
		fmt.Printf("Length mismatch: expected %d, got %d\n", len(expected), len(got))
		return false
	}

	for i := range expected {
		for key, expectedVal := range expected[i] {
			gotVal, exists := got[i][key]
			if !exists {
				fmt.Printf("Key %s does not exist in got[%d]\n", key, i)
				return false
			}

			// Compare values with type flexibility
			expectedValue := expectedVal["value"]
			gotValue := gotVal["value"]

			// Handle slice conversion
			var normalizedExpectedValue, normalizedGotValue interface{}

			// Convert interface{} slice to []uint8 or []byte if needed
			if expectedSlice, ok := expectedValue.([]interface{}); ok {
				convertedSlice := make([]uint8, len(expectedSlice))
				for j, v := range expectedSlice {
					if intVal, ok := v.(int); ok {
						convertedSlice[j] = uint8(intVal)
					}
				}
				normalizedExpectedValue = convertedSlice
			} else {
				normalizedExpectedValue = expectedValue
			}

			// Ensure gotValue is also a []uint8
			if gotSlice, ok := gotValue.([]uint8); ok {
				normalizedGotValue = gotSlice
			} else {
				normalizedGotValue = gotValue
			}

			// Convert numeric types to a common representation
			switch v := normalizedExpectedValue.(type) {
			case int:
				normalizedExpectedValue = int64(v)
			case int64:
				normalizedExpectedValue = v
			}

			switch v := normalizedGotValue.(type) {
			case int:
				normalizedGotValue = int64(v)
			case int64:
				normalizedGotValue = v
			}

			// Perform comparison
			if !reflect.DeepEqual(normalizedExpectedValue, normalizedGotValue) {
				fmt.Printf("Value mismatch for key %s\n", key)
				fmt.Printf("Expected value: %v (type: %T)\n", expectedValue, expectedValue)
				fmt.Printf("Got value:      %v (type: %T)\n", gotValue, gotValue)
				return false
			}
		}
	}
	return true
}

func compareResults(db *sqlx.DB, yamlFile string) (bool, error) {
	// Read and parse YAML file
	content, err := os.ReadFile(yamlFile)
	if err != nil {
		return false, fmt.Errorf("failed to read YAML file: %v", err)
	}

	var testFile TestFile
	if err := yaml.Unmarshal(content, &testFile); err != nil {
		return false, fmt.Errorf("failed to parse YAML: %v", err)
	}
	for _, queryTest := range testFile.Queries {
		rows, err := db.Queryx(queryTest.Query)
		if err != nil {
			return false, fmt.Errorf("query failed: %v", err)
		}
		defer rows.Close()

		var actualResults []map[string]map[string]interface{}

		// Get actual results with types
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return false, fmt.Errorf("scan failed: %v", err)
			}

			typedRow := make(map[string]map[string]interface{})
			for k, v := range row {
				valueMap := make(map[string]interface{})

				switch typedV := v.(type) {
				case int64:
					valueMap["value"] = typedV
					valueMap["type"] = "int64"
				case []uint8:
					valueMap["value"] = typedV
					valueMap["type"] = "[]uint8"
				default:
					valueMap["value"] = v
					valueMap["type"] = fmt.Sprintf("%T", v)
				}

				typedRow[k] = valueMap
			}
			actualResults = append(actualResults, typedRow)
		}

		// Use the new flexible comparison
		if !deepCompareResults(queryTest.Expected, actualResults) {
			return false, fmt.Errorf("queries did not match expected results")
		}
	}
	return true, nil
}

func normalizeResults(results []map[string]map[string]interface{}) []map[string]map[string]interface{} {
	normalized := make([]map[string]map[string]interface{}, len(results))
	for i, result := range results {
		normalizedResult := make(map[string]map[string]interface{})
		for key, value := range result {
			normalizedValue := make(map[string]interface{})

			// Normalize item_name byte slice to string
			if key == "item_name" {
				if byteSlice, ok := value["value"].([]byte); ok {
					normalizedValue["value"] = string(byteSlice)
				} else if byteSlice, ok := value["value"].([]interface{}); ok {
					// Convert []interface{} to string
					bytes := make([]byte, len(byteSlice))
					for j, v := range byteSlice {
						if val, ok := v.(int); ok {
							bytes[j] = byte(val)
						}
					}
					normalizedValue["value"] = string(bytes)
				} else {
					normalizedValue["value"] = value["value"]
				}
				normalizedValue["type"] = "string"
			} else {
				normalizedValue = value
			}

			normalizedResult[key] = normalizedValue
		}
		normalized[i] = normalizedResult
	}
	return normalized
}

func testQuery(db *sqlx.DB, query string, expectedResults []map[string]map[string]interface{}) error {
	rows, err := db.Queryx(query)
	if err != nil {
		return fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	var gotResults []map[string]map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			return fmt.Errorf("scan failed: %v", err)
		}

		// Create result with type information
		typedRow := make(map[string]map[string]interface{})
		for k, v := range row {
			valueMap := make(map[string]interface{})

			switch typedV := v.(type) {
			case []byte:
				valueMap["value"] = typedV
				valueMap["type"] = "[]byte"
			case int64:
				valueMap["value"] = typedV
				valueMap["type"] = "int64"
			default:
				valueMap["value"] = v
				valueMap["type"] = fmt.Sprintf("%T", v)
			}

			typedRow[k] = valueMap
		}
		gotResults = append(gotResults, typedRow)
	}

	// Normalize results before comparison
	normalizedExpected := normalizeResults(expectedResults)
	normalizedGot := normalizeResults(gotResults)

	// Use reflect.DeepEqual on normalized results
	if !reflect.DeepEqual(normalizedExpected, normalizedGot) {
		return fmt.Errorf("queries did not match expected results")
	}

	return nil
}
