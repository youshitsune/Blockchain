VERSION 0.8
FROM golang:alpine
WORKDIR /workdir

docker:
    RUN apk update && apk add libc6-compat
    RUN wget https://github.com/youshitsune/blockchain/releases/download/v1.0.0/blockchain
    RUN chmod +x blockchain
    ENTRYPOINT ["/workdir/blockchain"]
    SAVE IMAGE blockchain:latest

