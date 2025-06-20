FROM heroiclabs/nakama-pluginbuilder:3.27.0 AS builder

ENV GO111MODULE on
ENV CGO_ENABLED 1
ENV GOPRIVATE "voidexforge"

WORKDIR /backend
COPY . .

RUN go build --trimpath --mod=vendor --buildmode=plugin -o ./backend.so

FROM 680994843819.dkr.ecr.ap-southeast-2.amazonaws.com/nakama:3.27.0
#FROM heroiclabs/nakama:3.27.0
#FROM voidexforge:local

COPY --from=builder /backend/backend.so /nakama/data/modules
COPY --from=builder /backend/configs /nakama/data/modules/configs
COPY --from=builder /backend/local.yml /nakama/data/
COPY --from=builder /backend/production.yml /nakama/data/
COPY --from=builder /backend/configs.yaml /nakama/

# Add a healthcheck that doesn't require database connection during build
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:7350/ || exit 1