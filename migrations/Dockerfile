FROM postgres:15-alpine

# Copy migration SQL file
COPY init-schema.sql /migrations/init-schema.sql

# Copy entrypoint script
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
