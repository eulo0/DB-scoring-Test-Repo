FROM mariadb:latest

# Environment variables for MariaDB
ENV MYSQL_User=root
ENV MYSQL_ROOT_PASSWORD=rootpassword

COPY "schema.sql" /docker-entrypoint-initdb.d/2-schema.sql

# Set permissions
RUN chmod 644 /docker-entrypoint-initdb.d/*.sql

# Expose MySQL port
EXPOSE 3306

# mysql -u trump -p 5010