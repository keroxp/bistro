FROM golang:1.27-trixie
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
WORKDIR /app
COPY go.mod go.sum Makefile main.go /app/
COPY pkg /app/pkg
RUN go mod tidy
COPY pkg /app/pkg
RUN make main VERSION="$VERSION" COMMIT="$COMMIT" DATE="$DATE" && \
    chmod +x main && \
    mv ./main /bin/bistro
