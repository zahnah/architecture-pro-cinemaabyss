FROM golang:1.27-bookworm

ARG PROTOC_GEN_GO_VERSION=v1.36.12
ARG PROTOC_VERSION=35.1
ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends curl unzip && rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}

# protoc bundles the well-known types (google/protobuf/*.proto) in include/;
# they are needed for timestamp.proto.
RUN case "${TARGETARCH:-amd64}" in \
      amd64) arch=x86_64 ;; \
      arm64) arch=aarch_64 ;; \
      *) echo "unsupported arch ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
 && curl -fsSL -o /tmp/protoc.zip \
      "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${arch}.zip" \
 && unzip -q /tmp/protoc.zip -d /protoc \
 && cp /protoc/bin/protoc /usr/local/bin/protoc \
 && cp -r /protoc/include/. /usr/local/include/ \
 && chmod -R a+rX /usr/local/include

COPY generate-code.sh /usr/local/bin/generate-code

WORKDIR /work
ENTRYPOINT ["generate-code"]
