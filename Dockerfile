# Use minimal base image
FROM alpine:latest


WORKDIR /app

# Copy the executable
COPY myapp .

RUN chmod +x myapp

ARG MYSQL_PASSW
# Set default command
ENTRYPOINT ["myapp"]
