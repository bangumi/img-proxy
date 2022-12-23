FROM busybox AS busybox

ARG BUSYBOX_VERSION=1.31.0-i686-uclibc
ADD https://busybox.net/downloads/binaries/$BUSYBOX_VERSION/busybox_WGET /wget
RUN chmod a+x /wget

# need to use github ci ubuntu-20.04
FROM gcr.io/distroless/base-debian11

COPY --from=busybox /wget /usr/bin/wget

ENTRYPOINT ["/app/img-proxy"]

COPY /dist/img-proxy /app/img-proxy
