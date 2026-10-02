FROM powerman/dockerize@sha256:ffbc7a88b04f83e911145b1dc7010b4c3c9c5420cc6b30ca1763e6aeb245bcd1 AS dockerize

FROM gcr.io/distroless/base-debian13@sha256:389cad21f73e4c37b94ffe5b13736d5a92bd5bd3c6c6b38c2be1c881e14ba2bd

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy
