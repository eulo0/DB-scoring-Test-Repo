package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

func generateTestFile(db *sqlx.DB, queryFile, outputFile string) error {
	file, err := os.Open(queryFile)
	if err != nil {
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

		rows, err := db.Queryx(query)
		if err != nil {
			return fmt.Errorf("query failed: %v", err)
		}
		defer rows.Close()

		var results []map[string]map[string]interface{}
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return fmt.Errorf("scan failed: %v", err)
			}

			// Create result with type information
			typedRow := make(map[string]map[string]interface{})
			for k, v := range row {
				valueMap := make(map[string]interface{})

				// Explicitly handle known types
				switch typedV := v.(type) {
				case int64:
					valueMap["value"] = typedV
					valueMap["type"] = "int64" // Explicitly set to int64
				case []uint8:
					valueMap["value"] = typedV
					valueMap["type"] = "[]uint8" // Explicitly set to []uint8
				default:
					valueMap["value"] = v
					valueMap["type"] = fmt.Sprintf("%T", v)
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
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	if err := os.WriteFile(outputFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}
