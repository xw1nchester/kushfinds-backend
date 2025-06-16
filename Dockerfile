FROM golang:1.24-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum /.
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/kushfinds/main.go


FROM alpine AS build-release-stage

WORKDIR /

COPY --from=build-stage /app/main /main

EXPOSE 8080

CMD ["/main"]