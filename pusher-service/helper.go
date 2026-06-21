package main

import "time"

// CreatedFeedMessage representa el "Payload" de datos estructurados que viaja por la red.
// El microservicio Pusher lee este modelo desde el bus de eventos (NATS) para
// serializarlo y "empujarlo" directamente a los navegadores de los usuarios conectados.
type CreatedFeedMessage struct {
	// Type define el identificador de tipo del mensaje de forma explícita en el JSON.
	// Es muy útil en el frontend (JavaScript) para que un solo canal de WebSocket
	// pueda discernir qué tipo de objeto llegó y cómo renderizarlo (ej: switch(msg.type)).
	Type string `json:"type"`

	// ID único del feed (las KSUIDs que configuramos previamente).
	ID string `json:"id"`

	// Title contiene el encabezado de la noticia o publicación.
	Title string `json:"title"`

	// Description contiene el cuerpo o texto principal del feed.
	Description string `json:"description"`

	// CreatedAt guarda la estampa de tiempo exacta de la creación en formato RFC3339 para JSON.
	CreatedAt time.Time `json:"created_at"`
}

// newCreatedFeedMessage actúa como función constructora (Factory Method).
// Centraliza y automatiza la inicialización de mensajes para garantizar que el campo
// "Type" siempre lleve el literal "created_feed" de manera consistente en todo el sistema.
func newCreatedFeedMessage(id, title, description string, createdAt time.Time) *CreatedFeedMessage {
	return &CreatedFeedMessage{
		Type:        "created_feed", // Hardcodificado aquí para evitar errores tipográficos manuales
		ID:          id,
		Title:       title,
		Description: description,
		CreatedAt:   createdAt,
	}
}

/*
Este struct representa exactamente el formato del mensaje que el servicio va a decodificar desde NATS para luego retransmitirlo en formato JSON plano a través de los WebSockets que maneja tu componente Client.
*/
