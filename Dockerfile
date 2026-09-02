FROM powerman/dockerize@sha256:ffbc7a88b04f83e911145b1dc7010b4c3c9c5420cc6b30ca1763e6aeb245bcd1 AS dockerize

FROM gcr.io/distroless/base-debian13@sha256:9ef50bca108839d5986e4d84b7f7b2d79024c9293b7c35b162c6c55485bd5868

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy
