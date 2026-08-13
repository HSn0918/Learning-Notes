# 多阶段构建: 一阶段编静态二进制, 二阶段塞进 distroless.
FROM golang:1.26 AS builder
WORKDIR /workspace
COPY go.mod ./
RUN go mod download || true
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/hami-mac .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/hami-mac /hami-mac
USER 0:0
ENTRYPOINT ["/hami-mac"]
