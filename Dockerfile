FROM golang:1.23.3 AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV REDIS_ADDR=redis:6379

WORKDIR /app
COPY . .

RUN go mod tidy

RUN mkdir -p /app/test_results && \
  if go test ./internal/app/... -v > /app/test_results/unit_test_results.txt; then \
  echo "Unit tests passed"; \
  else \
  cat /app/test_results/unit_test_results.txt && exit 1; \
  fi


RUN go build -o main ./cmd/server/main.go && \
  chmod +x main

RUN go test -c -o e2e.test ./internal/tests/e2e && \
  go test -c -o integration.test ./internal/tests/integration && \
  chmod +x e2e.test && \
  chmod +x integration.test

RUN go build -o redis_tui ./internal/infrastructure/tui/redis_tui.go && \
  chmod +x redis_tui

FROM alpine:3.19 AS app
RUN apk add --no-cache upx=4.2.1-r0  \
  curl=8.9.1-r1

WORKDIR /app

COPY --from=builder /app/main /app/main
COPY --from=builder /app/e2e.test /app/e2e.test
COPY --from=builder /app/integration.test /app/integration.test
COPY --from=builder /app/test_results /app/test_results
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY .env /app/.env

RUN upx --best --lzma /app/main -o /app/main_compressed && \
  mv /app/main_compressed /app/main

ENTRYPOINT ["/app/main"]
EXPOSE 8080


FROM alpine:3.19 AS tui

RUN apk add --no-cache \
  ncurses=6.4_p20231125-r0 \
  docker-cli=25.0.5-r1 \
  curl=8.9.1-r1

WORKDIR /app

COPY --from=builder /app/redis_tui /app/redis_tui
COPY .env /app/.env

RUN export TERM=xterm-256color && \
  infocmp > /dev/null

ENV TERM=xterm-256color

ENTRYPOINT ["/app/redis_tui"]

