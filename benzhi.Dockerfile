# 评测专用（勿覆盖项目自带 Dockerfile）
FROM golang:1.21

ENV GOTOOLCHAIN=local

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build ./...

CMD ["bash"]
