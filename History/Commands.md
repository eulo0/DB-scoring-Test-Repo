# Commands are here to make it easy for copying and pasting

# Build the image
docker build -t test .

# Run the container
docker run -d --name test-db -p 5010:3306 test-db

# Removing the container
docker remove test-db

# Connecting to database example
mysql -u root -p -P 5010 -h localhost           

# ^^^^^ password is skibidigronk