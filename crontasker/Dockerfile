# 使用 Go 的官方 alpine 版本
FROM golang:alpine as builder

# 設置工作目錄
WORKDIR /home/crontasker

# 複製 crontasker 的內容
COPY crontasker/ .

# 複製 herofishingGoModule 到 /home/herofishingGoModule
COPY herofishingGoModule/ /home/herofishingGoModule/

# 設置 Go 模組環境變數
ENV GO111MODULE=on

# 建置 crontasker
RUN go mod tidy
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o crontasker .

# 複製執行檔
FROM alpine
WORKDIR /app
COPY --from=builder /home/crontasker/crontasker /app/

# 執行
CMD ["/app/crontasker"]
