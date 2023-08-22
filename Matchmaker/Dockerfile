FROM golang:alpine as go
WORKDIR /go/src/matchmaker
ENV GO111MODULE=on

COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o matchmaker .

CMD ["/go/src/matchmaker/matchmaker"]