FROM powerman/dockerize@sha256:e2c34e00f1f8a8886aea2508ef4d680343f90302d3dbf4d3c08cab43351b07cf AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:9dce90e688a57e59ce473ff7bc4c80bc8fe52d2303b4d99b44f297310bbd2210

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy
