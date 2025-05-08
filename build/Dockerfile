FROM heroiclabs/nakama-pluginbuilder:3.26.0 AS builder

ENV GO111MODULE on
ENV CGO_ENABLED 1
ENV GOPRIVATE "voidexforge"

WORKDIR /voidexforge
COPY . .

RUN go build --trimpath --mod=vendor --buildmode=plugin -o ./backend.so main.go

#FROM 680994843819.dkr.ecr.ap-southeast-2.amazonaws.com/ondigames/voidexforgeserver:dev
FROM heroiclabs/nakama:3.26.0
COPY --from=builder /voidexforge/backend.so /nakama/data/modules
COPY --from=builder /voidexforge/*.lua /nakama/data/modules/
COPY --from=builder /voidexforge/build/*.js /nakama/data/modules/build/
COPY --from=builder /voidexforge/local.yml /nakama/data/