FROM golang:1.26.3-alpine

WORKDIR /app

# 安装必要的工具（包括 Python 和 pip）
RUN apk add --no-cache git python3 py3-pip

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 安装 Python 依赖
RUN pip3 install --no-cache-dir curl-cffi pybase64 --break-system-packages

# 构建应用
RUN go build -o main ./cmd/server/

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./main"]
