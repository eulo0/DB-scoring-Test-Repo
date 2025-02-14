# DIRECTORIES
    dbfiles: holds dockerfile for creating db container along with sql files
    main: tests scoring functionality
    queries: holds files used by a service for scoring
    tools: generates files for "queries" directory given 2 sql files: "users.sql" and "schema&seed.sql"

# LIMITATIONS

At the moment, the code is only used for testing reads. Testing write operations is being worked on

# Build Process (starting from root directory):
cd dbfiles
docker build -t minecraft .

# For testing if read matches expected values does score...
cd ../main
go run .

# For testing scoring mismatches, just edit any value in trump.yaml