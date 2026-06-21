package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader configura las reglas para transformar una petición HTTP normal en un WebSocket de larga duración.
var upgrader = websocket.Upgrader{
	// CheckOrigin: true desactiva la validación de CORS. Es ideal para entornos de desarrollo
	// o cuando Nginx (API Gateway) se encarga de filtrar los orígenes.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub actúa como el despachador de eventos (Event Dispatcher) centralizado.
// MEJORA: Eliminamos el Mutex. Toda la mutación de estado ocurre dentro de una única Goroutine
// secuencial en el método Run(), garantizando la seguridad de hilos (Thread-Safety) por diseño.
type Hub struct {
	clients    map[*Client]bool // MEJORA: Cambiado slice por mapa para búsquedas y borrados en tiempo O(1)
	register   chan *Client
	unregister chan *Client
	broadcast  chan interface{} // MEJORA: Canal dedicado para procesar broadcasts de forma no bloqueante
}

// NewHub inicializa las estructuras de datos y canales del Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan interface{}, 256), // ¡Corregido aquí con llaves!
	}
}

// Run es el bucle de eventos asíncrono que corre perpetuamente en su propia Goroutine.
// Al procesar todas las operaciones sobre 'hub.clients' secuencialmente en este select,
// evitamos condiciones de carrera sin usar Mutex bloqueantes.
func (hub *Hub) Run() {
	for {
		select {
		case client := <-hub.register:
			hub.onConnect(client)

		case client := <-hub.unregister:
			hub.onDisconnect(client)

		case message := <-hub.broadcast:
			hub.onBroadcast(message)
		}
	}
}

// Broadcast es un método público y seguro que puede ser invocado desde los hilos de NATS.
// Simplemente inyecta el mensaje en el canal de procesamiento para que el bucle Run() lo maneje.
func (hub *Hub) Broadcast(message interface{}) {
	hub.broadcast <- message
}

// onConnect registra internamente al cliente en el mapa de conexiones activas.
func (hub *Hub) onConnect(client *Client) {
	log.Println("Client connected:", client.socket.RemoteAddr())
	client.id = client.socket.RemoteAddr().String()
	hub.clients[client] = true
}

// onDisconnect limpia de forma segura los recursos del cliente y lo remueve del mapa.
func (hub *Hub) onDisconnect(client *Client) {
	// Verificamos si el cliente realmente existe en nuestro mapa antes de proceder a borrar
	if _, exists := hub.clients[client]; exists {
		log.Println("Client disconnected:", client.socket.RemoteAddr())
		delete(hub.clients, client)

		// Invocamos el método Close() que mejoramos en el paso anterior (protegido con sync.Once)
		client.Close()
	}
}

// onBroadcast ejecuta la distribución masiva de bytes serializados a todos los sockets.
func (hub *Hub) onBroadcast(message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("failed to marshal broadcast message: %v", err)
		return
	}

	// Iteramos eficientemente sobre las llaves de nuestro mapa de clientes
	for client := range hub.clients {
		select {
		// Intentamos inyectar los bytes en el canal de salida del cliente
		case client.outbound <- data:
			// Envío exitoso, el canal del cliente tenía espacio libre.

		default:
			// MEJORA CRÍTICA (Slow Client Protection): Si el canal 'outbound' de este cliente
			// en específico está lleno debido a lag en su red, el caso 'default' se activa de inmediato.
			// Esto evita que un solo usuario lento congele la entrega de mensajes a todo el resto del sistema.
			log.Printf("Client %s slow connection. Dropping buffer and disconnecting.", client.id)

			// Removemos al cliente problemático y liberamos sus sockets
			delete(hub.clients, client)
			client.Close()
		}
	}
}

// HandleWebSocket es el punto de entrada HTTP (Endpoint Handler).
// Eleva el protocolo de la conexión y le delega el control del socket al Hub.
func (hub *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade connection: %v", err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}

	client := NewClient(hub, socket)

	// Enviamos al cliente al canal de registro para que sea procesado de forma segura por el Hub
	hub.register <- client

	// Encendemos la Goroutine de escritura exclusiva para este socket.
	// La lectura de mensajes provenientes del navegador (si se requiere) se iniciaría en otra goroutine separada.
	go client.Write()
}

/*
¡Llegamos al cerebro del Pusher Service! El Hub es el encargado de centralizar y orquestar todas las conexiones WebSocket del sistema. Su rol es crítico: cuando llega un evento de NATS, el Hub toma ese mensaje y hace un Broadcast (difusión masiva) hacia todos los clientes web conectados.
*/
