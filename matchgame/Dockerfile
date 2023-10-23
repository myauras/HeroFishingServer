# 使用Go特定版本來建置購執行檔
FROM golang:1.21.0 as builder

# 設置工作目錄
WORKDIR /home/matchgame

# 設置 Go 模組環境變數
ENV GO111MODULE=on

# 複製 matchgame 的內容
COPY matchgame/ .

# 複製 herofishingGoModule 到 /home/herofishingGoModule
COPY herofishingGoModule/ /home/herofishingGoModule/

# 下載建置需要的模組
RUN go mod tidy
RUN go mod download
# 建置 matchgame
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o matchgame .


# 使用 alpine版本建置server image
FROM golang:alpine

# 建立一個server user
RUN adduser -D -u 1000 serverUser
# 將建置環境的執行檔複製到server環境中
COPY --from=builder /home/matchgame/matchgame /home/matchgame/
# 複製 Keys 到 /home/Keys
COPY Keys/ /home/Keys/

RUN chown -R serverUser /home/matchgame/matchgame && \
    chmod o+x /home/matchgame/matchgame

USER 1000

# 設置執行指令
ENTRYPOINT ["/home/matchgame/matchgame"]




