package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"platzi.com/go/cqrs/database"
	"platzi.com/go/cqrs/events"
	"platzi.com/go/cqrs/repository"
	"platzi.com/go/cqrs/search"
)

// Config define la estructura de las variables de entorno para el lado de la Lectura (Query Model).
type Config struct {
	PostgresHost         string `envconfig:"POSTGRES_HOST" default:"postgres"` // MEJORA: Host dinámico para desarrollo o producción
	PostgresDB           string `envconfig:"POSTGRES_DB"`
	PostgresUser         string `envconfig:"POSTGRES_USER"`
	PostgresPassword     string `envconfig:"POSTGRES_PASSWORD"`
	NatsAddress          string `envconfig:"NATS_ADDRESS"`
	ElasticsearchAddress string `envconfig:"ELASTICSEARCH_ADDRESS"`
}

// newRouter define los endpoints del API orientados exclusivamente a la lectura de datos.
func newRouter() (router *mux.Router) {
	router = mux.NewRouter()
	// GET /feeds: Obtiene la lista relacional plana (fallback/respaldo)
	router.HandleFunc("/feeds", listFeedsHandler).Methods(http.MethodGet)
	// GET /search?q=... : Realiza la búsqueda full-text optimizada en Elasticsearch
	router.HandleFunc("/search", searchHandler).Methods(http.MethodGet)
	return
}

func main() {
	// 1. CARGA DE CONFIGURACIÓN
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	// 2. CONEXIÓN AL WRITE MODEL (POSTGRES) - Solo para lectura directa/listados analíticos
	addr := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresHost,
		cfg.PostgresDB,
	)
	repo, err := database.NewPostgresRepository(addr)
	if err != nil {
		log.Fatal(err)
	}
	repository.SetRepository(repo)

	// 3. CONEXIÓN AL READ MODEL (ELASTICSEARCH)
	es, err := search.NewElastic(fmt.Sprintf("http://%s", cfg.ElasticsearchAddress))
	if err != nil {
		log.Fatal(err)
	}
	search.SetSearchRepository(es)

	// Registramos el cierre limpio del cliente de búsquedas
	defer func() {
		log.Println("Cerrando repositorio de Elasticsearch...")
		search.Close()
	}()

	// 4. CONEXIÓN AL BUS DE EVENTOS (NATS) Y REGISTRO DEL CONSUMIDOR
	n, err := events.NewNats(fmt.Sprintf("nats://%s", cfg.NatsAddress))
	if err != nil {
		log.Fatal(err)
	}

	// CORRECCIÓN CRÍTICA: Añadido context.Background() para cumplir con la firma concurrente actualizada de NATS
	err = n.OnCreatedFeed(context.Background(), onCreatedFeed)
	if err != nil {
		log.Fatal(err)
	}
	events.SetEventStore(n)

	// Registramos el cierre limpio de las suscripciones y sockets de NATS
	defer func() {
		log.Println("Cancelando suscripciones y cerrando conexión con NATS...")
		events.Close()
	}()

	// 5. ARRANQUE DEL SERVIDOR HTTP
	router := newRouter()

	log.Printf("Servidor de Consultas (query-service) escuchando en el puerto :8080...")

	// CORRECCIÓN: Capturamos el error de manera controlada para permitir que la función finalice
	// de manera natural y los bloques 'defer' limpien la red de Docker.
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Println("El servidor HTTP se ha detenido debido a un error:", err)
	}
}

/*
Este main.go es el encargado de arrancar todo el ecosistema de consulta: inicializa las conexiones a Postgres, a Elasticsearch y lo más importante, se suscribe asíncronamente al Bus de Eventos (NATS) para que cuando llegue un nuevo feed, el handler onCreatedFeed lo indexe en tiempo real.
*/
