FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY main /app/main
RUN chmod +x /app/main
WORKDIR /app
CMD ["/app/main"]
