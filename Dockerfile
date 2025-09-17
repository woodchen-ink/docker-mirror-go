# Use pre-built binary
FROM alpine:latest

# Copy the pre-built binary based on target architecture
ARG TARGETARCH
COPY docker-mirror-go-linux-${TARGETARCH} ./docker-mirror-go

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Move binary to working directory
RUN mv /docker-mirror-go ./docker-mirror-go

# Make it executable
RUN chmod +x ./docker-mirror-go

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./docker-mirror-go"]