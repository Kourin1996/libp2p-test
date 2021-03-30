FROM tetafro/golang-gcc:1.15-alpine

WORKDIR $GOPATH/src/github.com/Kourin1996/libp2p-test
ADD . $GOPATH/src/github.com/Kourin1996/libp2p-test

RUN apk update
RUN apk add --no-cache git make bash curl
RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
