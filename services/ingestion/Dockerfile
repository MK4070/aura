FROM golang:1.26.3-alpine AS builder

WORKDIR /app

# For tokenizer
RUN mkdir /empty-tmp && chmod 777 /empty-tmp

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/uploader ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/processor ./cmd/worker

FROM scratch

COPY --from=builder /empty-tmp /tmp

COPY --from=builder /bin/uploader /uploader
COPY --from=builder /bin/processor /processor