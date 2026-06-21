# Arquitectura CQRS con Event Sourcing en Go

Este proyecto es una implementación completa y robusta de un sistema de publicaciones (**Feeds**) utilizando el patrón de diseño **CQRS (Command Query Responsibility Segregation)** y **Event Sourcing** en el lenguaje Go. El sistema separa estrictamente las operaciones de escritura de las de lectura, comunicándose de forma asíncrona mediante un bus de eventos y notificando a los usuarios finales en tiempo real.

---

## 🏗️ Arquitectura del Sistema

El sistema se divide en tres microservicios principales y componentes de infraestructura de soporte independientes:

1. **Feed Service (Command Model - Escritura):** Recibe las peticiones de creación de contenido. Valida, asigna identificadores únicos ordenados cronológicamente (**KSUID**) y persiste el estado puro en una base de datos relacional (PostgreSQL). Posteriormente, propaga el evento de creación.
2. **Query Service (Query Model - Lectura):** Escucha el bus de eventos en segundo plano. Cuando se notifica un nuevo registro, proyecta los datos indexándolos en un motor de búsqueda full-text (Elasticsearch). Atiende las consultas de búsqueda masiva con tolerancia ortográfica (*fuzziness*).
3. **Pusher Service (Real-Time Service):** Se suscribe al bus de eventos y actúa como despachador masivo (*Hub Multiplexor*), distribuyendo las actualizaciones en tiempo real a los navegadores de los clientes mediante conexiones concurrentes de WebSockets.

---

## 📁 Estructura del Proyecto

```text
arquit_eventos_CQRS/
├── database/            # Configuración, scripts SQL (up.sql) y Dockerfile de PostgreSQL
├── events/              # Implementación abstracta y concreta del bus de eventos (NATS)
├── models/              # Modelos de datos del dominio compartidos (Feed, Message structs)
├── repository/          # Interfaces y fachadas de abstracción de datos globales (Inversión de dependencias)
├── search/              # Implementación del cliente optimizado para Elasticsearch
├── feed-service/        # Punto de entrada y controladores HTTP para el Command Model
├── query-service/       # Punto de entrada y controladores HTTP para el Query Model
├── pusher-service/      # Gestión del Hub concurrente y sockets de tiempo real
├── nginx.conf           # Configuración del API Gateway dinámico por método HTTP
├── Dockerfile           # Multi-stage Dockerfile optimizado para compilación cruzada en Go
└── docker-compose.yaml  # Orquestación de infraestructura local de contenedores

## 📦 Módulos y Tecnologías Utilizadas

* **Go (v1.26):** Lenguaje base de alto rendimiento concurrente.
* **NATS Server (v2.14):** Broker de mensajería asíncrono de ultra baja latencia para la propagación del Event Bus.
* **Elasticsearch (v7.x / v6.x):** Motor analítico de lectura estructurada para proyecciones full-text rápidas.
* **PostgreSQL (v16):** Base de datos relacional y transaccional para el almacenamiento de comandos.
* **Nginx (v1.27):** API Gateway que divide dinámicamente el tráfico hacia Escritura (`POST /feeds`) o Lectura (`GET /feeds`).
* **Gorilla Mux & Sockets:** Enrutamiento HTTP homogéneo y WebSockets para persistencia en tiempo real.
* **Segmentio KSUID:** Identificadores únicos globales legibles y ordenados en el tiempo.
* **Kelseyhightower Envconfig:** Gestión limpia y tipada de variables de entorno inyectadas en Docker.

---

## 🚀 Cómo Ejecutar el Proyecto

El proyecto está completamente preparado para desplegarse de manera automatizada con un solo comando gracias a las construcciones por etapas (*multi-stage*) de Docker:

### Requisitos previos:
* Tener instalado Docker y Docker Compose.

### Inicialización del clúster:
En la raíz del proyecto ejecuta:

```bash
docker-compose up -d --build

Nginx expondrá públicamente el puerto `80`. Puedes probar el circuito de la siguiente manera:

1. **Crear Feed (Comando):** Envía una petición `POST` a `http://localhost/feeds` utilizando Postman o cURL con el siguiente cuerpo JSON:
   ```json
   {
       "title": "Mi curso de Go",
       "description": "Mi aplicación hecha en Go"
   }
*El sistema responderá con un estado `201 Created` y el objeto persistido con su respectiva KSUID.*

2. **Buscar Feed (Lectura):** Envía una petición `GET` a `http://localhost/search?q=curso`. Nginx redirigirá la solicitud al modelo de lectura optimizado, el cual consultará los índices de Elasticsearch utilizando búsquedas con tolerancia ortográfica (*fuzziness*).

3. **Escuchar Eventos (Tiempo Real):** Conecta un cliente de pruebas de WebSockets (como la extensión de Postman o `wscat`) a la dirección `ws://localhost/ws`. Cada vez que realices un comando `POST`, el `pusher-service` empujará de forma instantánea y concurrente el payload del evento a todos los clientes conectados.

---

## 🛠️ Mantenimiento y Logs

Para monitorear el flujo de eventos, la sincronización de NATS y las inserciones de las bases de datos en tiempo real, puedes seguir los logs unificados del clúster ejecutando:

```bash
docker-compose logs -f

Si deseas ver el comportamiento de un microservicio específico (por ejemplo, el de lectura):

```bash
docker-compose logs -f query
