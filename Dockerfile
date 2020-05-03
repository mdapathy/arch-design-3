FROM golang:1.14 as build

RUN apt-get update && apt-get install -y ninja-build

# TODO: Змініть на власну реалізацію системи збірки
RUN go get -u github.com/mdapathy/arch-design-2/build/cmd/boodtdma

WORKDIR /go/src/practice-3
COPY . .

RUN CGO_ENABLED=0 boodtdma

# ==== Final image ====
FROM alpine:3.11
WORKDIR /opt/practice-3
COPY entry.sh ./
COPY --from=build /go/src/practice-3/out/bin/* ./
ENTRYPOINT ["/opt/practice-3/entry.sh"]
CMD ["server"]
