package events

import (
	"context"

	"platzi.com/go/cqrs/models"
)

// EventStore define el contrato abstracto (Interface) para el manejo de eventos en el sistema.
// Cualquier tecnología de mensajería (NATS, Kafka, RabbitMQ) que use el proyecto
// debe implementar obligatoriamente estos métodos para interactuar con la lógica de CQRS.
type EventStore interface {
	// Close cierra de forma limpia la conexión física con el servidor de mensajería.
	Close()

	// PublishCreatedFeed publica el evento de que un nuevo feed ha sido creado en el Bus de Eventos.
	// Es el disparador principal que usa el servicio de Comandos (Escritura).
	PublishCreatedFeed(ctx context.Context, feed *models.Feed) error

	// SubscribeCreatedFeed retorna un canal de Go (read-only channel `<-chan`) por el cual
	// el microservicio suscriptor puede escuchar ráfagas continuas de mensajes del tipo CreatedFeedMessage.
	SubscribeCreatedFeed(ctx context.Context) (<-chan CreatedFeedMessage, error)

	// OnCreatedFeed es un enfoque alternativo basado en Callbacks (handlers).
	// Registra una función ejecutuable 'f' que se activará automáticamente cada vez que llegue un evento.
	OnCreatedFeed(ctx context.Context, f func(CreatedFeedMessage)) error
}

// -----------------------------------------------------------------------------
// PATRÓN SINGLETON / FAÇADE (Fachada)
// Las siguientes funciones exponen el EventStore a nivel de paquete.
// Esto evita tener que pasar la variable del cliente NATS de estructura en estructura,
// permitiendo usar llamadas directas como `events.PublishCreatedFeed(...)`.
// -----------------------------------------------------------------------------

// Variable interna privada del paquete que guarda la implementación concreta elegida (ej. NATS).
var eventStore EventStore

// SetEventStore es el punto de inyección de dependencias.
// Se invoca en el main.go al arrancar el servidor para asignar el cliente real de NATS.
func SetEventStore(store EventStore) {
	eventStore = store
}

// Close actúa como proxy para cerrar el bus de datos de forma global.
func Close() {
	eventStore.Close()
}

// PublishCreatedFeed encapsula la publicación. El contexto (ctx) propaga cancelaciones
// o timeouts a lo largo del pipeline de mensajería.
func PublishCreatedFeed(ctx context.Context, feed *models.Feed) error {
	return eventStore.PublishCreatedFeed(ctx, feed)
}

// SubscribeCreatedFeed expone el canal de lectura a los microservicios de Query o Pusher.
func SubscribeCreatedFeed(ctx context.Context) (<-chan CreatedFeedMessage, error) {
	return eventStore.SubscribeCreatedFeed(ctx)
}

// OnCreatedFeed asocia la función de callback de manera global.
func OnCreatedFeed(ctx context.Context, f func(CreatedFeedMessage)) error {
	return eventStore.OnCreatedFeed(ctx, f)
}

/*
Este archivo es fundamental porque define el contrato de abstracción (Interface) para tu Bus de Eventos.

Al usar este patrón, aíslas tu lógica de negocio de la tecnología real (NATS). Si el día de mañana decides cambiar NATS por Apache Kafka o RabbitMQ, tu código de negocio no sufrirá ningún cambio, solo tendrás que crear una nueva estructura que implemente esta interfaz.
*/
