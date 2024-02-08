# syntax=docker/dockerfile:1

FROM golang:1.21

# Set destination for COPY
WORKDIR /application
COPY . /application
# Download Go modules


# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy

RUN go install github.com/matr-builder/matr

# Build
RUN matr build

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 8080

CMD ["./build/server", "-port", "8080"]
