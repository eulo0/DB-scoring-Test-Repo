package main

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jmoiron/sqlx"
	_ "gopkg.in/yaml.v3"
	"tutorial.sqlc.dev/app/tools"
)

// resolveRelativePath helps resolve paths relative to the current directory
func resolveRelativePath(relativePath string) string {
	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return ""
	}

	// Resolve the full path
	fullPath := filepath.Join(currentDir, relativePath)

	// Convert to absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		fmt.Println("Error resolving path:", err)
		return ""
	}

	return absPath
}

func main() {
	// Use the helper function to resolve paths
	sqlFilePath := resolveRelativePath("../../dbfiles/users.sql")
	userQueriesFilePath := resolveRelativePath("../../user_queries/sample.txt")
	creds_serviceQueriesFilePath := resolveRelativePath("../../service_queries/mysqlfiles/users.txt")
	queries_serviceQueriesFilePath := resolveRelativePath("../../service_queries/mysqlfiles/queries.yaml")
	// Call functions with discovered file paths
	tools.ExtractUsersPasswords(sqlFilePath, creds_serviceQueriesFilePath)
	err := tools.GenerateTestFile(userQueriesFilePath, queries_serviceQueriesFilePath)
	// Assuming generateTestFile needs a database connection and query file
	//err := generateTestFile(nil, queriesFilePath, resolveRelativePath("queries.yaml"))
	//if err != nil {
	//	fmt.Println("Error generating test file:", err)
	//}
	if err != nil {
		fmt.Printf("Error generating test file: %v\n", err)
	}
}
