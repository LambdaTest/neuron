FROM neuron-base as builder

# create a working directory
COPY . /neuron
WORKDIR /neuron

# Build binary
RUN GOARCH=amd64 GOOS=linux go build -ldflags="-w -s" -o nu
# Uncomment only when build is highly stable. Compress binary.
# RUN strip --strip-unneeded ts
# RUN upx ts
# use a minimal alpine image
FROM alpine:3.12
# add ca-certificates in case you need them
RUN apk update && apk add ca-certificates libc6-compat && rm -rf /var/cache/apk/*
# set working directory
WORKDIR /root
# copy the binary from builder
COPY --from=builder /neuron/nu .
# copy email template from builder
COPY --from=builder /neuron/static/mailtemplates  ./static/mailtemplates
# run the binary
ENTRYPOINT  ["./nu"]