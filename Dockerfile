FROM golang:alpine as builder

RUN mkdir /app
WORKDIR /app

COPY . .

RUN go mod download
RUN GOOS=linux go build -o ctweb *.go

# Run container
FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN mkdir /app
WORKDIR /app
COPY --from=builder /app/ctweb .
ENV BATCH_SIZE=100
ENV BATCH_INTERVAL=10000

# HERE PORT has to match with POST_URL localhost:PORT to test with inbuild posturl api
ENV PORT=8080
ENV POST_URL=http://localhost:8080/postLog

CMD ["./ctweb"]