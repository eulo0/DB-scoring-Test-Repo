# DIRECTORIES
    dbfiles: holds dockerfile for creating db container along with sql files
    main: tests scoring functionality
    queries: holds files used by a service for scoring
    tools: generates files for "queries" directory given 2 sql files: "users.sql" and "schema&seed.sql"

# SETUP



# TOOLS GUIDE

If cloned from the repo, there is no need to use tools. however if you want to test out tools, delete all files stored in queries and then do the following: 


# Build Process (starting from root directory):
cd dbfiles
docker build -t minecraft .

cd ../tools
go run sqlQuery_generate.go

cd ../main