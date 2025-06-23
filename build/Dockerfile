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

# Add entrypoint script
RUN echo '#!/bin/sh\n\
set -e\n\
/nakama/nakama migrate up --database.address postgres://root:Ondi2025@128.199.91.28:26257/test\n\
exec /nakama/nakama --config /nakama/data/production.yml --database.address postgres://root:Ondi2025@128.199.91.28:26257/test "$@"' > /entrypoint.sh && \
    chmod +x /entrypoint.sh

# Add a healthcheck that doesn't require database connection during build
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:7350/ || exit 1

ENTRYPOINT ["/entrypoint.sh"]