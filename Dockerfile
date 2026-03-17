FROM golang:alpine AS build

ARG APP_VERSION=dev

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go tool templ generate
RUN CGO_ENABLED=1 go build -ldflags="-w -s -X catgoose/dothog/internal/version.Version=${APP_VERSION}" -o /dothog .

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /dothog /usr/local/bin/dothog

ENV SERVER_LISTEN_PORT=3000
EXPOSE 3000

ENTRYPOINT ["dothog"]
