package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	elastic "github.com/elastic/go-elasticsearch/v7"
	"platzi.com/go/cqrs/models"
)

// ElasticSearchRepository implementa la persistencia optimizada para consultas de lectura estructuradas.
// En CQRS, este componente no maneja lógica transaccional, su único fin es resolver búsquedas full-text de forma veloz.
type ElasticSearchRepository struct {
	client *elastic.Client
}

// NewElastic inicializa el cliente oficial configurando las direcciones del cluster.
func NewElastic(url string) (*ElasticSearchRepository, error) {
	client, err := elastic.NewClient(elastic.Config{
		Addresses: []string{url},
	})
	if err != nil {
		return nil, err
	}

	return &ElasticSearchRepository{
		client: client,
	}, nil
}

// Close se incluye para cumplir la interfaz genérica de repositorios en caso de ser necesario.
func (r *ElasticSearchRepository) Close() {
	// El cliente v7 de elasticsearch maneja internamente un pool de conexiones HTTP persistente,
	// por lo que no requiere un método explícito de cierre de socket.
}

// IndexFeed indexa (inserta o actualiza) un documento en Elasticsearch.
// Se invoca desde el servicio de Queries al reaccionar a un evento proveniente de NATS.
func (r *ElasticSearchRepository) IndexFeed(ctx context.Context, feed models.Feed) error {
	body, err := json.Marshal(feed)
	if err != nil {
		return err
	}

	// Index almacena el documento usando la KSUID de Go como el ID nativo del documento en el índice "feeds".
	_, err = r.client.Index(
		"feeds",
		bytes.NewReader(body),
		r.client.Index.WithDocumentID(feed.ID),
		r.client.Index.WithContext(ctx),
		// "wait_for" bloquea la respuesta hasta que el cambio sea visible para las búsquedas (Consistencia eventual controlada).
		r.client.Index.WithRefresh("wait_for"),
	)
	return err
}

// SearchFeed realiza una búsqueda elástica de texto utilizando similitud fonética y aproximada (Fuzzy Search).
func (r *ElasticSearchRepository) SearchFeed(ctx context.Context, query string) ([]models.Feed, error) {
	var buf bytes.Buffer

	// Construcción del DSL (Domain Specific Language) de Elasticsearch.
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":            query,
				"fields":           []string{"title", "description"}, // Busca coincidencias en ambas columnas
				"fuzziness":        "AUTO",                           // 'AUTO' ajusta la tolerancia de errores ortográficos según el largo del texto.
				"cutoff_frequency": 0.0001,                           // Ignora palabras extremadamente comunes de forma dinámica.
			},
		},
	}

	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		return nil, err
	}

	// Ejecuta la consulta HTTP POST al endpoint /feeds/_search
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex("feeds"),
		r.client.Search.WithBody(&buf),
		r.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	// CRÍTICO: El Body debe cerrarse siempre para liberar la conexión HTTP de vuelta al pool.
	defer res.Body.Close()

	// Verifica si el código de estado HTTP devuelto por Elasticsearch es un error (4xx o 5xx)
	if res.IsError() {
		return nil, errors.New("elasticsearch error: " + res.String())
	}

	// Estructura contenedora intermedia para mapear la respuesta JSON nativa de Elasticsearch
	var eRes struct {
		Hits struct {
			Hits []struct {
				Source json.RawMessage `json:"_source"` // Mantiene el fragmento en bytes puros para evitar doble parsing
			} `json:"hits"`
		} `json:"hits"`
	}

	// Decodificamos el cuerpo de la respuesta directamente sobre nuestra estructura limpia 'eRes'
	if err := json.NewDecoder(res.Body).Decode(&eRes); err != nil {
		return nil, err
	}

	// MEJORA EXTRA DE RENDIMIENTO: Conocemos exactamente cuántos resultados (hits) devolvió la consulta.
	// Hacemos un 'make' con capacidad pre-asignada (0 elementos iniciales, pero espacio reservado para len(hits)).
	// Esto evita re-asignaciones continuas de memoria en el slice subyacente de Go dentro del bucle.
	totalHits := len(eRes.Hits.Hits)
	feeds := make([]models.Feed, 0, totalHits)

	// Iteramos sobre la lista de aciertos (hits) encontrados por el motor de búsqueda
	for i := 0; i < totalHits; i++ {
		var feed models.Feed
		// Deserializamos directamente los bytes planos del campo '_source' hacia nuestro modelo definitivo
		if err := json.Unmarshal(eRes.Hits.Hits[i].Source, &feed); err == nil {
			feeds = append(feeds, feed)
		}
	}

	return feeds, nil
}

/*

 */
