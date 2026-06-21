package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"platzi.com/go/cqrs/events"
	"platzi.com/go/cqrs/models"
	"platzi.com/go/cqrs/repository"
	"platzi.com/go/cqrs/search"
)

// onCreatedFeed es el "Event Consumer" o suscriptor de NATS.
// Se ejecuta de manera asíncrona cada vez que el Command Service publica que un feed fue creado.
// Su única misión es realizar la proyección del Read Model persistiendo los datos en Elasticsearch.
func onCreatedFeed(m events.CreatedFeedMessage) {
	feed := models.Feed{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
	}

	// MEJORA: Añadimos un timeout de 5 segundos para proteger el hilo de ejecución de NATS
	// en caso de que Elasticsearch experimente alta latencia o caídas puntuales.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // Siempre liberamos los recursos del contexto al finalizar

	if err := search.IndexFeed(ctx, feed); err != nil {
		log.Printf("failed to index feed in elasticsearch: %v", err)
	}
}

// listFeedsHandler retorna la lista cruda de feeds desde la base de datos relacional.
// Sirve como consulta de apoyo o fallback directo de proyecciones planas.
func listFeedsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	feeds, err := repository.ListFeeds(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// CORRECCIÓN: El Content-Type DEBE ir estrictamente antes de WriteHeader()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(feeds)
}

// searchHandler es la joya de la corona del Read Model.
// Resuelve búsquedas complejas full-text de forma instantánea consultando a Elasticsearch.
func searchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extraemos el parámetro de búsqueda desde la URL (ej: /search?q=golang)
	query := r.URL.Query().Get("q")
	if len(query) == 0 {
		http.Error(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	feeds, err := search.SearchFeed(ctx, query)
	if err != nil {
		// CORRECCIÓN CRÍTICA: Se añade el 'return' que faltaba para evitar que el flujo continúe
		// e intente codificar datos nulos tras haber escrito un error HTTP 500.
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// CORRECCIÓN: Estructuración correcta de cabeceras HTTP de respuesta JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(feeds)
}

/*
¡Este archivo es el núcleo del Query Service (Servicio de Lecturas)! Su responsabilidad en la arquitectura es doble: atender las consultas de búsqueda de la API leyendo desde Elasticsearch y reaccionar asíncronamente a los eventos de NATS para mantener el índice actualizado (Proyección).
*/
