FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/demo/ ./cmd/demo/
RUN CGO_ENABLED=0 go build -o /demo ./cmd/demo

FROM scratch
COPY --from=build /demo /demo
USER 10001
ENV APP_ADDR=:8000
ENTRYPOINT ["/demo"]
