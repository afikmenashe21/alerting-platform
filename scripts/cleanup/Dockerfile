FROM alpine:3.19

# Install Redis CLI, PostgreSQL client, bash
RUN apk add --no-cache redis postgresql-client bash

# Copy cleanup script
COPY cleanup.sh /cleanup.sh
RUN chmod +x /cleanup.sh

ENTRYPOINT ["/cleanup.sh"]
