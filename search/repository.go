package search

import (
	"context"

	"platzi.com/go/cqrs/models"
)

// SearchRepository define el contrato abstracto (Interface) para el Modelo de Lectura (Read Model).
// Al igual que con el Event Bus, esta abstracción permite que la lógica de la API de consultas
// no dependa directamente de Elasticsearch. Si en el futuro decides migrar a Algolia, Meilisearch
// o una base de datos de grafos, solo tendrás que crear un nuevo struct que cumpla esta interfaz.
type SearchRepository interface {
	// Close asegura la liberación ordenada de cualquier cliente de red HTTP o TCP persistente.
	Close()

	// IndexFeed inserta o actualiza el documento indexado para que sea inmediatamente rastreable.
	IndexFeed(ctx context.Context, feed models.Feed) error

	// SearchFeed ejecuta la consulta full-text sobre las proyecciones y devuelve las coincidencias.
	SearchFeed(ctx context.Context, query string) ([]models.Feed, error)
}

// -----------------------------------------------------------------------------
// PATRÓN SINGLETON / FAÇADE (Fachada)
// Estas funciones y variables exponen el repositorio a nivel global del paquete,
// permitiendo realizar llamadas limpias como `search.SearchFeed(...)` desde tus handlers.
// -----------------------------------------------------------------------------

// Variable interna privada que almacena la implementación concreta elegida (ej. ElasticSearchRepository).
var repo SearchRepository

// SetSearchRepository es el punto de inyección de dependencias.
// Se invoca en el main.go del microservicio de consultas (query-service) al iniciar el servidor.
func SetSearchRepository(r SearchRepository) {
	repo = r
}

// Close actúa como proxy para invocar de manera segura el método de cierre global.
func Close() {
	repo.Close()
}

// IndexFeed encapsula la indexación pasándole el Context (ctx) para controlar cancelaciones de peticiones web.
func IndexFeed(ctx context.Context, feed models.Feed) error {
	return repo.IndexFeed(ctx, feed)
}

// SearchFeed encapsula la búsqueda elástica y devuelve el set final de datos tipados.
func SearchFeed(ctx context.Context, query string) ([]models.Feed, error) {
	return repo.SearchFeed(ctx, query)
}

/*
Este componente es el que conectará directamente con el consumidor de eventos. Cuando el query-service escuche a través de NATS un evento created_feed, tomará el payload del mensaje y llamará inmediatamente a search.IndexFeed(ctx, feed) para mantener tu índice de Elasticsearch sincronizado en tiempo real con la base de datos de escritura.
*/
