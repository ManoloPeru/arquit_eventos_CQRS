package models

import "time"

// Feed representa la entidad de datos principal para el flujo de información (Read/Write Model).
// En una arquitectura CQRS, esta estructura funciona como el "Data Transfer Object" (DTO)
// plano que se expone a través de la API y se guarda en la base de datos de proyecciones.
type Feed struct {
	// ID único del feed. Al ser de tipo 'string', encaja perfectamente con las KSUID
	// que genera la librería de Segmentio que instalamos previamente.
	// El tag json:"id" le dice a Go cómo serializar/deserializar este campo en las APIs REST.
	ID string `json:"id"`

	// Title almacena el título o cabecera del feed de noticias o eventos.
	Title string `json:"title"`

	// Description contiene el cuerpo, texto o metadata larga del feed.
	Description string `json:"description"`

	// CreatedAt guarda la marca de tiempo exacta de cuándo ocurrió el evento o se creó el registro.
	// El tipo time.Time de Go se mapea de forma nativa con el tipo TIMESTAMP de PostgreSQL.
	CreatedAt time.Time `json:"created_at"`
}

/*
Tener los tags de JSON en minúsculas (json:"id", json:"created_at") es una excelente práctica para tu API pública. Como vimos en las preguntas de teoría al inicio, esto garantiza que cuando devuelvas un JSON o lo serialices para enviar un mensaje, mantenga el estándar de la web web (camelCase o snake_case) en lugar de exportar las llaves con la primera letra mayúscula de Go.
*/
