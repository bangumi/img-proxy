FROM powerman/dockerize AS dockerize

FROM gcr.io/distroless/base-debian12

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy
