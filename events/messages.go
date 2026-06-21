package events

import "time"

// Message define la interfaz base para cualquier evento en el sistema.
// Exigir el método Type() permite implementar un patrón de "Polimorfismo de Eventos".
// Gracias a esto, el despachador de NATS o tus consumidores pueden leer el tipo de mensaje
// en texto plano antes de intentar deserializar el JSON al struct específico.
type Message interface {
	Type() string
}

// CreatedFeedMessage representa el "Payload" (cuerpo de datos) del evento específico
// que se dispara inmediatamente después de que un feed se guarda con éxito en la base de datos de escritura.
// Nota que replica exactamente los campos del modelo original para transferir el estado completo.
type CreatedFeedMessage struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Type implementa de forma implícita la interfaz 'Message' para la estructura CreatedFeedMessage.
// Retorna un identificador único en string ("created_feed"). Esto funciona como la "etiqueta"
// o metadato del evento, ideal para estructuras select/case en los consumidores que procesan múltiples tipos de eventos.
func (m CreatedFeedMessage) Type() string {
	return "created_feed"
}

/*
Este componente establece el contrato para todos los eventos que viajan a través de tu Event Bus (NATS), garantizando que el sistema sea extensible y tipado de forma segura
*/
