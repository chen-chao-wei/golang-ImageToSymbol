FROM golang:1.11.2-alpine
WORKDIR /go-test
ADD . /go-test
RUN cd /go-test && go build
EXPOSE 3333
ENTRYPOINT ./go-test