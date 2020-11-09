FROM golang:1.15-buster
WORKDIR /go-test
ADD . /go-test
RUN cd /go-test && go get github.com/nfnt/resize && go build
EXPOSE 3333
CMD ["./go-test"]
