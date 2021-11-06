FROM golang:1.16 as build

WORKDIR /nbuild
COPY . .
RUN go mod vendor && go build ./cmd/nbuild/main.go

FROM ghcr.io/void-linux/xbps-src-masterdir:v20211105RC01-x86_64
WORKDIR /opt/voidlinux/nbuild
COPY --from=build /nbuild/main ./nbuild
COPY docker-entrypoint.sh .
ENV NBUILD_BIND=:8080 \
        NBUILD_BITCASK_PATH=/opt/voidlinux/nbuild/bitcask \
        NBUILD_COMPONENTS=graph
ENTRYPOINT ["/opt/voidlinux/nbuild/docker-entrypoint.sh"]
