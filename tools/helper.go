package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

type ColumnValue struct {
	Value interface{} `yaml:"value"`
	Type  string      `yaml:"type"`
}

type QueryResult struct {
	Query    string                              `yaml:"query"`
	Expected []map[string]map[string]interface{} `yaml:"expected"`
}

type TestFile struct {
	Queries []QueryResult `yaml:"queries"`
}

func ExtractUsersPasswords(inputFilename, outputFilename string) {
	// Open the input SQL file
	file, err := os.Open(inputFilename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Regex to match CREATE USER statements
	re := regexp.MustCompile(`CREATE USER '([^']+)'@'[^']*' IDENTIFIED BY '([^']+)'`)

	var users []string
	scanner := bufio.NewScanner(file)

	// Read file line by line
	for scanner.Scan() {
		line := scanner.Text()

		// Find matches in the line
		matches := re.FindAllStringSubmatch(line, -1)

		// Process matches
		for _, match := range matches {
			if len(match) == 3 {
				// Format: username:password
				users = append(users, fmt.Sprintf("%s:%s", match[1], match[2]))
			}
		}
	}

	// Check for any scanning errors
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Write to output text file
	err = os.WriteFile(outputFilename, []byte(strings.Join(users, "\n")), 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Printf("Users and passwords have been extracted to %s\n", outputFilename)

	// Print extracted users
	for _, user := range users {
		fmt.Println(user)
	}
}

func establishDBconnection() (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Second)
	defer cancel()

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		"trump", "skibidigronk", "localhost", 5010, "minecraft")
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

// Rename this to be more helpful later
func GenerateTestFile(queryFile, outputFile string) error {
	fmt.Printf("Generating test file from %s to %s\n", queryFile, outputFile)

	db, err := establishDBconnection()
	if err != nil {
		fmt.Printf("Database connection error: %v\n", err)
		return err
	}
	defer db.Close()

	file, err := os.Open(queryFile)
	if err != nil {
		fmt.Printf("Failed to open query file: %v\n", err)
		return fmt.Errorf("failed to open query file: %v", err)
	}
	defer file.Close()

	var testFile TestFile
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		query := strings.TrimSpace(scanner.Text())
		if query == "" {
			continue
		}

		fmt.Printf("Executing query: %s\n", query)

		rows, err := db.Queryx(query)
		if err != nil {
			fmt.Printf("Query failed: %v\n", err)
			return fmt.Errorf("query failed: %v", err)
		}
		defer rows.Close()
		var results []map[string]map[string]interface{}
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return fmt.Errorf("scan failed: %v", err)
			}

			typedRow := make(map[string]map[string]interface{})
			for k, v := range row {
				valueMap := make(map[string]interface{})

				// Convert types during YAML generation
				switch val := v.(type) {
				case []uint8:
					// Keep []uint8 as is
					valueMap["value"] = val
					valueMap["type"] = "[]uint8"
				case int64:
					valueMap["value"] = val
					valueMap["type"] = "int64"
				case time.Time:
					valueMap["value"] = val.UTC()
					valueMap["type"] = "time.Time"
				case float64:
					valueMap["value"] = val
					valueMap["type"] = "float64"
				default:
					valueMap["value"] = val
					valueMap["type"] = fmt.Sprintf("%T", val)
				}
				typedRow[k] = valueMap
			}
			results = append(results, typedRow)
		}

		testFile.Queries = append(testFile.Queries, QueryResult{
			Query:    query,
			Expected: results,
		})
	}

	output, err := yaml.Marshal(testFile)
	if err != nil {
		fmt.Printf("Failed to marshal YAML: %v\n", err)
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	if err := os.WriteFile(outputFile, output, 0644); err != nil {
		fmt.Printf("Failed to write output file: %v\n", err)
		return fmt.Errorf("failed to write output file: %v", err)
	}

	fmt.Printf("Successfully generated test file at %s\n", outputFile)
	return nil
}
