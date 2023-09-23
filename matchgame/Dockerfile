FROM golang:1.15.13 as builder
WORKDIR /go/src/matchgame

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o server .

FROM alpine:3.11

RUN adduser -D -u 1000 server
COPY --from=builder /go/src/game/server /home/server/server
# COPY key_no_upload /home/server/key_no_upload
RUN chown -R server /home/server && \
    chmod o+x /home/server/server

USER 1000
ENTRYPOINT ["/home/server/server"]
