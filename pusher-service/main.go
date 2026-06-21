package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux" // Importado para estandarizar el enrutamiento con los otros servicios
	"github.com/kelseyhightower/envconfig"
	"platzi.com/go/cqrs/events"
)

// Config almacena las variables de entorno inyectadas por Docker para el Pusher Service.
type Config struct {
	NatsAddress string `envconfig:"NATS_ADDRESS"`
}

func main() {
	// 1. CARGA DE CONFIGURACIÓN
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	// 2. INICIALIZACIÓN DEL HUB DE WEBSOCKETS
	hub := NewHub()

	// Arrancamos el bucle del select del Hub en segundo plano (Goroutine asíncrona).
	// Esto permite que el hilo principal continúe configurando NATS y levante el servidor HTTP.
	go hub.Run()

	// 3. CONEXIÓN AL EVENT BUS (NATS) Y ESCUCHA EN TIEMPO REAL
	n, err := events.NewNats(fmt.Sprintf("nats://%s", cfg.NatsAddress))
	if err != nil {
		log.Fatal(err)
	}

	// CORRECCIÓN 1: Se agrega context.Background() para cumplir con la firma concurrente de la interfaz.
	// CORRECCIÓN 2: Se remueve el parámetro redundante ', nil' de hub.Broadcast ya que el Hub optimizado no lo requiere.
	err = n.OnCreatedFeed(context.Background(), func(m events.CreatedFeedMessage) {
		// Encapsulamos los datos en nuestro mensaje de salida con el tag "type": "created_feed"
		// y lo enviamos al canal de distribución del Hub.
		hub.Broadcast(newCreatedFeedMessage(m.ID, m.Title, m.Description, m.CreatedAt))
	})
	if err != nil {
		log.Fatal(err)
	}
	events.SetEventStore(n)

	// Aseguramos la desconexión ordenada del bus al finalizar o interrumpir el contenedor
	defer func() {
		log.Println("Cancelando suscripciones de WebSockets y cerrando NATS...")
		events.Close()
	}()

	// 4. CONFIGURACIÓN DEL ENRUTADOR (API GATEWAY COMPATIBLE)
	// MEJORA: Cambiado de http.HandleFunc a un router explícito de Gorilla Mux para mantener
	// la consistencia con feed-service y query-service, facilitando el balanceo en Nginx.
	router := mux.NewRouter()
	router.HandleFunc("/ws", hub.HandleWebSocket)

	log.Printf("Servidor de Notificaciones en Tiempo Real (pusher-service) escuchando en el puerto :8080...")

	// CORRECCIÓN 3: Evitamos pasarle log.Fatal directamente a la ejecución del servidor
	// para no romper la ejecución de los bloques 'defer' superiores si el puerto se encuentra ocupado.
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Println("El servidor HTTP de WebSockets se ha detenido debido a un error:", err)
	}
}

/*
¡Llegamos al main.go definitivo del Pusher Service, Manuel! Este archivo cierra el circuito completo de tiempo real: lee la configuración, arranca la goroutine del Hub para gestionar los WebSockets, monta el servidor HTTP en el endpoint /ws y se conecta a NATS para retransmitir instantáneamente cada feed que se cree.
*/
