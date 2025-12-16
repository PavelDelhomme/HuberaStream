FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata ffmpeg
COPY main /app/main
RUN chmod +x /app/main
WORKDIR /app
RUN mkdir -p /app/storage/videos /app/storage/thumbnails
CMD ["/app/main"]
