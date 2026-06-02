FROM powerman/dockerize@sha256:e2c34e00f1f8a8886aea2508ef4d680343f90302d3dbf4d3c08cab43351b07cf AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:58695f439f772a00009c8f6be4c183f824c1f556d74b313c30900f167e4772f8

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy
