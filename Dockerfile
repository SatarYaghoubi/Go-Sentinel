# ---- build stage ----
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/sentinel ./cmd/sentinel

# ---- run stage ----
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/sentinel /sentinel
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/sentinel"]
