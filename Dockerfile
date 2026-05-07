FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /llmprobe .

FROM alpine:3.21

COPY --from=build /llmprobe /usr/local/bin/llmprobe

ENTRYPOINT ["llmprobe", "mcp"]
