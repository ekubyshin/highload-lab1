ARG GOLANG_IMAGE=golang:1.26.8-alpine3.24
ARG ALPINE_IMAGE=alpine:3.24.3

FROM ${GOLANG_IMAGE} AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS=-mod=readonly

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
RUN go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM ${ALPINE_IMAGE} AS runtime
WORKDIR /app
USER 10001:10001

FROM runtime AS dashboard-generator
COPY --from=build /out/dashboard-generator /app/dashboard-generator
ENTRYPOINT ["/app/dashboard-generator"]

FROM runtime AS stack-init
COPY --from=build /out/stack-init /app/stack-init
ENTRYPOINT ["/app/stack-init"]

FROM runtime AS app
COPY --from=build /out/app /app/app
ENTRYPOINT ["/app/app"]
