FROM golang:1.19-bullseye AS builder
WORKDIR /tmp/clae

ADD . /tmp/clae/

RUN go build -ldflags="-extldflags=-static" -tags sqlite_omit_load_extension -o target/clae main.go

FROM gcr.io/distroless/static-debian11
COPY --from=builder /tmp/clae/target/clae /
COPY --from=builder /tmp/clae/html        /html

ENTRYPOINT ["/clae"]