FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/egresados_service .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/egresados_service ./egresados_service
COPY conf ./conf
COPY swagger ./swagger
EXPOSE 8081
ENTRYPOINT ["./egresados_service"]
