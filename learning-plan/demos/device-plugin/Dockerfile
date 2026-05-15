# 多阶段构建: 第一阶段编译静态二进制, 第二阶段用 scratch 作为最小镜像。
# Device Plugin 通常以 DaemonSet 部署, 镜像越小启动越快、攻击面越小。

FROM golang:1.26 AS builder
WORKDIR /workspace

# 先拷 go.mod / go.sum 利用 layer 缓存。
COPY go.mod ./
# COPY go.sum ./   # 首次执行 go mod tidy 后取消注释
RUN go mod download || true

COPY *.go ./
# CGO_ENABLED=0 + -ldflags '-s -w' 出一个 ~10MB 的静态二进制, 可塞进 scratch。
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/fake-device-plugin .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/fake-device-plugin /fake-device-plugin
USER 0:0
ENTRYPOINT ["/fake-device-plugin"]
