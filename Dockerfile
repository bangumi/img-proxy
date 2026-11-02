FROM powerman/dockerize@sha256:06f81150637e09b7e73d25801f7b7592a70a3567c649b9f120081f5543321520 AS dockerize

FROM gcr.io/distroless/base-debian13@sha256:075a33d9c49026ca7ee72738b4c4a1caea55d0f44227356e46a611a211050e20

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy
