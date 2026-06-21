# syntax=docker/dockerfile:1

FROM node:24-bookworm AS web-build

WORKDIR /src/web/app

ARG VITE_PUBLIC_API_BASE_URL=
ENV VITE_PUBLIC_API_BASE_URL=${VITE_PUBLIC_API_BASE_URL}

COPY web/app/package.json web/app/pnpm-lock.yaml ./
RUN corepack enable \
    && pnpm install --frozen-lockfile

COPY web/app ./
RUN pnpm build \
    && test -f build/client/index.html

FROM golang:1.25.7-bookworm AS build

ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go build -trimpath -ldflags="-s -w" -o /out/go-scaffold-server ./cmd/aoi

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 app \
    && useradd --system --uid 10001 --gid app --home-dir /app --shell /usr/sbin/nologin app

WORKDIR /app

COPY --from=build /out/go-scaffold-server /app/go-scaffold-server
COPY configs/config.example.yaml /app/configs/config.example.yaml
COPY deploy/config.production.example.yaml /app/configs/config.yaml
COPY configs/locales /app/configs/locales
COPY --from=web-build /src/web/app/build/client /app/web/app/build/client

RUN mkdir -p /app/data /app/logs \
    && chown -R app:app /app

USER app

EXPOSE 9999

ENV RIN_CONFIG_PATH=/app/configs/config.yaml

ENTRYPOINT ["/app/go-scaffold-server"]
CMD ["server", "--config=/app/configs/config.yaml"]
