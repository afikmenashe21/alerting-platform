FROM postgres:15-alpine

# Copy migration SQL files
COPY init-schema.sql /migrations/init-schema.sql
COPY add-composite-indexes.sql /migrations/add-composite-indexes.sql
COPY add-counts-cache.sql /migrations/add-counts-cache.sql

# Copy entrypoint script
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
