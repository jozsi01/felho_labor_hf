# Use minimal base image
FROM alpine:latest

# Copy the executable
COPY myapp /usr/local/bin/myapp

ARG MYSQL_PASSW
# Set default command
ENTRYPOINT ["myapp"]
