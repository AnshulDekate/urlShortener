FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o main .


FROM scratch AS final
ENV TZ=UTC
COPY --from=builder /app/main /main
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/main"]
