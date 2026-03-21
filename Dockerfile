FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /unirelease .

FROM alpine:3.20
RUN apk add --no-cache git
COPY --from=builder /unirelease /usr/local/bin/unirelease
ENTRYPOINT ["unirelease"]
