package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/segmentio/ksuid"
	"platzi.com/go/cqrs/events"
	"platzi.com/go/cqrs/models"
	"platzi.com/go/cqrs/repository" // Importa la fachada de persistencia SQL (Write Model)
)

// createFeedRequest define la estructura estricta del Payload JSON que esperamos del cliente (Command Data)
type createFeedRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// createFeedHandler implementa la lógica del controlador HTTP para la creación de un feed (Command).
func createFeedHandler(w http.ResponseWriter, r *http.Request) {
	// 1. DESERIALIZACIÓN: Validamos que el cuerpo de la petición sea un JSON legible
	var req createFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2. ORQUESTACIÓN DE DATOS: Preparamos la entidad con marcas de tiempo en formato UTC
	createdAt := time.Now().UTC()

	// Generamos una KSUID para tener un ID único y ordenado cronológicamente desde el origen
	id, err := ksuid.NewRandom()
	if err != nil {
		// CORRECCIÓN: Ajustado el mensaje para corresponder al dominio de feeds
		http.Error(w, "failed to generate record id", http.StatusInternalServerError)
		return
	}

	feed := models.Feed{
		ID:          id.String(),
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   createdAt,
	}

	// 3. PERSISTENCIA TRANSACCIONAL (Write Model): Guardamos el estado puro en PostgreSQL
	if err := repository.InsertFeed(r.Context(), &feed); err != nil {
		// CORRECCIÓN: Agregado el 'return' para evitar que intente publicar en NATS si la BD falló
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. EVENT SOURCING / PROPAGACIÓN: Publicamos el evento asíncronamente en NATS (Event Bus)
	// Usamos r.Context() para que, si el cliente cancela la petición HTTP a mitad de camino,
	// se pueda propagar la cancelación en la red si el driver lo soporta.
	if err := events.PublishCreatedFeed(r.Context(), &feed); err != nil {
		// No cortamos el flujo de la petición si NATS falla aquí (dependiendo de la consistencia requerida),
		// pero dejamos registro en el log del sistema para auditoría.
		log.Printf("failed to publish created feed event: %v", err)
	}

	// 5. RESPUESTA HTTP: Retornamos la proyección final al cliente
	// MEJORA: Las cabeceras como Content-Type DEBEN ir estrictamente antes de WriteHeader()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Envía el estado HTTP 201

	// Codificamos el struct 'feed' directamente hacia la respuesta web
	json.NewEncoder(w).Encode(feed)
}

/*
Este es el handler HTTP encargado de procesar los Commands (Escrituras) dentro de tu arquitectura CQRS: recibe la petición, escribe el estado en Postgres y propaga el evento asíncronamente a NATS.
*/
