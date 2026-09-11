FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/Team254/cheesy-arena-lite/version.Version=${VERSION}" \
    -o /out/cyber-arena .
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /out/cyber-arena /app/cyber-arena
COPY static/ /app/static/
COPY templates/ /app/templates/
COPY font/ /app/font/
COPY schedules/ /app/schedules/
COPY --from=build --chown=nonroot:nonroot /out/data /data

ENV ARENA_MODE=cloud \
    ARENA_PORT=9080 \
    ARENA_DB_PATH=/data/event.db

VOLUME /data
EXPOSE 9080
USER nonroot:nonroot
ENTRYPOINT ["/app/cyber-arena"]
