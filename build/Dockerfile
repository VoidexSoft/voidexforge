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
COPY --from=builder /backend/configs.yaml /nakama/