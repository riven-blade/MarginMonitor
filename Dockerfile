FROM alpine:latest

WORKDIR /app

COPY ./app /app/app

RUN chmod +x /app/app

CMD ["/app/app"]