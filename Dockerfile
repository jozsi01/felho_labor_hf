FROM alpine:latest

WORKDIR /app

COPY myapp /app/myapp

RUN chmod +x /app/myapp

ARG MYSQL_PASSW

ENTRYPOINT ["/app/myapp"]

