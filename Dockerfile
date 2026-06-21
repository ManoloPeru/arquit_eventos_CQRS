# =============================================================================
# ETAPA 1: COMPILACIÓN (Builder)
# =============================================================================
ARG GO_VERSION=1.26-alpine
FROM golang:${GO_VERSION} AS builder

# Actualización de Red: Instalamos git y certificados de seguridad
RUN apk add --no-cache git ca-certificates openssh-client && update-ca-certificates

# Configuración del proxy oficial y rápido
ENV GOPROXY=https://proxy.golang.org,direct

WORKDIR /src

# Copiamos los manifiestos de tus dependencias
COPY go.mod go.sum ./

# Descarga limpia de librerías
RUN go mod download

# Copiamos todo tu código fuente
COPY . .

# Compilamos tus tres microservicios concurrentes
RUN go install ./...

# =============================================================================
# ETAPA 2: IMAGEN DE PRODUCCIÓN (Runtime)
# =============================================================================
FROM alpine:3.20

# Heredamos los certificados de seguridad
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /usr/bin

# Copiamos los binarios puros (feed-service, query-service, pusher-service)
COPY --from=builder /go/bin/ .

# Nota de Integración con tu docker-compose.yaml:
# Como tu compose define parámetros 'command: "feed-service"' o 'command: "query-service"',
# Docker sabrá exactamente cuál de los tres binarios guardados en este directorio arrancar
# para cada contenedor asignado. ¡La automatización perfecta!

#Utiliza un patrón avanzado de ingeniería en contenedores conocido como Multi-stage Build (Construcción por etapas). Su gran ventaja es que utiliza un contenedor pesado con todo el SDK de Go para compilar tus ejecutables, y luego desecha todo ese peso para meter únicamente tus binarios finales dentro de una imagen ultra ligera de Alpine.