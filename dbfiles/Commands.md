# Commands are here to make it easy for copying and pasting. This is for building/interacting with db hosted by container

# Build the image
docker build -t minecraft-db .

# Run the container
docker run -d --name minecraft-db -p 5010:3306 minecraft-db

# Removing the container
docker remove minecraft-db

# Connecting to database example. The password is skibidigronk
mysql -u trump -p -P 5010 -h localhost           
