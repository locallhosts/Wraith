FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /wraith .

FROM python:3.11-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /wraith /usr/local/bin/wraith
COPY engine-python/requirements.txt engine-python/requirements.txt
RUN pip install --no-cache-dir -r engine-python/requirements.txt
COPY rules ./rules
COPY engine-python ./engine-python
EXPOSE 8080
ENTRYPOINT ["wraith", "serve"]
