FROM powerman/dockerize:0.17.0 AS binary

# need to use github ci ubuntu-20.04
FROM gcr.io/distroless/base-debian11

COPY --from=binary /usr/local/bin/dockerize /usr/bin/dockerize

ENTRYPOINT ["/app/img-proxy"]

COPY /dist/img-proxy /app/img-proxy

HEALTHCHECK --interval=30s \
            --timeout=5s \
            CMD ["/usr/bin/dockerize", "-timeout", "1s", "-wait", "http://127.0.0.1:8000/health", "-exit-code", "1"]
