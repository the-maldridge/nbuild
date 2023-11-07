ARG LIBC=glibc
FROM golang:1.21 as build

WORKDIR /nbuild
COPY . .
RUN go mod vendor && go build ./cmd/nbuild/main.go

FROM ghcr.io/void-linux/void-buildroot-${LIBC}:20231006R1 AS image
WORKDIR /opt/voidlinux/nbuild
COPY --from=build /nbuild/main ./nbuild
COPY docker-entrypoint.sh .
ENV NBUILD_BIND=:8080 \
        NBUILD_BITCASK_PATH=/opt/voidlinux/nbuild/bitcask \
        NBUILD_COMPONENTS=graph
ENTRYPOINT ["/opt/voidlinux/nbuild/docker-entrypoint.sh"]
