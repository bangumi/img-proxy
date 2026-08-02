FROM powerman/dockerize@sha256:717d6a9be2bc0a8420e2a59855b5eabf6f52bbabf3bd14d101392a42229e9954 AS dockerize

FROM gcr.io/distroless/base-debian13@sha256:f4a335ca209e1d2ee873102c17c389ad0142e3d5b21aee2817e9cc9c01d87d20

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy
