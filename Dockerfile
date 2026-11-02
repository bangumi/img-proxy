FROM powerman/dockerize@sha256:ffbc7a88b04f83e911145b1dc7010b4c3c9c5420cc6b30ca1763e6aeb245bcd1 AS dockerize

FROM gcr.io/distroless/base-debian13@sha256:075a33d9c49026ca7ee72738b4c4a1caea55d0f44227356e46a611a211050e20

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy
