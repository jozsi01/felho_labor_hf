FROM ubuntu:latest

COPY myapp/detection_program /app/myapp

RUN echo "Listing contents of /app:" && ls -l /app

RUN chmod +x /app/myapp

ARG MYSQL_PASSW
ARG DATABASE_ADDR
ARG DATABASE_PORT

ENTRYPOINT ["/app/myapp"]

