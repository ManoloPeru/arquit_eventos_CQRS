package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig" // ¡Nuestra librería corregida funcionando al 100%!
	"platzi.com/go/cqrs/database"
	"platzi.com/go/cqrs/events"
	"platzi.com/go/cqrs/repository"
)

// Config define las variables de entorno que tu contenedor Docker inyectará.
// Gracias a envconfig, se mapean automáticamente basándose en los tags.
type Config struct {
	PostgresHost     string `envconfig:"POSTGRES_HOST" default:"postgres"` // MEJORA: Evita el hardcoding del host en Docker
	PostgresDB       string `envconfig:"POSTGRES_DB"`
	PostgresUser     string `envconfig:"POSTGRES_USER"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD"`
	NatsAddress      string `envconfig:"NATS_ADDRESS"`
}

// newRouter inicializa y configura las rutas HTTP para el servicio de Comandos (Escritura).
func newRouter() (router *mux.Router) {
	// Inicializa una nueva instancia del enrutador de Gorilla Mux
	router = mux.NewRouter()

	// Mapea el endpoint POST /feeds al handler que procesa las inserciones y eventos
	router.HandleFunc("/feeds", createFeedHandler).Methods(http.MethodPost)
	return
}

func main() {
	// 1. CARGA DE CONFIGURACIÓN DE ENTORNO
	var cfg Config
	// Process lee el entorno del sistema operativo. El primer parámetro vacío ""
	// significa que no usará un prefijo (buscará directamente "POSTGRES_DB" en vez de "APP_POSTGRES_DB").
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err) // Aquí sí es seguro usar Fatal porque aún no hay defers declarados
	}

	// 2. CONEXIÓN E INYECCIÓN DEL REPOSITORIO (WRITE MODEL)
	// MEJORA: Usamos cfg.PostgresHost dinámicamente en lugar de la palabra estática 'postgres'
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
	// Inyectamos la conexión concreta de Postgres en la fachada global del paquete repository
	repository.SetRepository(repo)

	// 3. CONEXIÓN E INYECCIÓN DEL EVENT BUS (NATS)
	n, err := events.NewNats(fmt.Sprintf("nats://%s", cfg.NatsAddress))
	if err != nil {
		log.Fatal(err)
	}
	// Inyectamos el cliente real de NATS en la fachada global del paquete events
	events.SetEventStore(n)

	// CRÍTICO: Registramos el cierre ordenado de conexiones.
	// Al usar log.Println o control manual al final, aseguramos que el runtime ejecute esta línea.
	defer func() {
		log.Println("Cerrando recursos de red de manera ordenada...")
		events.Close()
	}()

	// 4. INICIALIZACIÓN DEL SERVIDOR WEB
	router := newRouter()

	log.Printf("Servidor de Comandos (feed-service) escuchando en el puerto :8080...")

	// CORRECCIÓN: Almacenamos el error en una variable en lugar de pasárselo a log.Fatal directo.
	// Esto garantiza que cuando el servidor se apague o falle, la función main() termine de forma natural
	// y el bloque 'defer' de arriba se ejecute limpiamente, evitando conexiones huérfanas en tu Docker network.
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Println("El servidor HTTP se ha detenido debido a un error:", err)
	}
}

/*
Este es el main.go de tu microservicio de Comandos / Escritura (feed-service). Aquí es donde envconfig (nuestra librería estrella) entra en acción mapeando el entorno, se inyectan las dependencias en las fachadas globales y se enciende el servidor HTTP.
*/
