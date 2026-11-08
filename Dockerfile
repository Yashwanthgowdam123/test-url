FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /myurls . && mkdir /data

FROM scratch
COPY --from=build /myurls /myurls
COPY --from=build --chown=65532:65532 /data /data
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD ["/myurls","-health"]
ENTRYPOINT ["/myurls"]
