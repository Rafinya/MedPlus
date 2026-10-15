FROM golang:alpine AS build
WORKDIR /app
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN go mod download
WORKDIR /app
COPY . .
WORKDIR /app/backend
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /medplus-pos .

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /medplus-pos /app/medplus-pos
COPY web /app/web
EXPOSE 8080
CMD ["/app/medplus-pos"]
