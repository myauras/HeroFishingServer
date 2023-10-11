# 使用 Go 的官方 alpine 版本
FROM golang:alpine as go

# 設置工作目錄
WORKDIR /home/matchmaker

# 設置 Go 模組環境變數
ENV GO111MODULE=on

# 複製 matchmaker 的內容
COPY matchmaker/ .

# 複製 herofishingGoModule 到 /home/herofishingGoModule
COPY herofishingGoModule/ /home/herofishingGoModule/

# 複製 Keys 到 /go/src/Keys
COPY Keys/ /home/

# 建置 matchmaker
RUN go mod tidy
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o matchmaker .

# 設置執行指令
CMD ["/home/matchmaker/matchmaker"]
