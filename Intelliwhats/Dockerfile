FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /out/intelliwhats .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata ffmpeg
COPY --from=build /out/intelliwhats /usr/local/bin/intelliwhats
RUN mkdir -p /app/login /app/temp
WORKDIR /app
ENTRYPOINT ["/usr/local/bin/intelliwhats"]
