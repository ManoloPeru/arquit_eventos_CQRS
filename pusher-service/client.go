package main

import (
	"sync" // Importado para manejar concurrencia segura sin pánicos

	"github.com/gorilla/websocket"
)

// Client modela y administra el ciclo de vida de una conexión WebSocket activa.
// Actúa como el puente final que "empuja" las actualizaciones de feeds en tiempo real al frontend.
type Client struct {
	hub      *Hub
	id       string
	socket   *websocket.Conn
	outbound chan []byte
	once     sync.Once // MEJORA: Evita pánicos por doble cierre del canal outbound
}

// NewClient inicializa la entidad del cliente y prepara el canal con búfer
func NewClient(hub *Hub, socket *websocket.Conn) *Client {
	return &Client{
		hub:    hub,
		socket: socket,
		// Tip pro: Es recomendable darle un pequeño buffer (ej: 256) para que ráfagas cortas
		// de eventos de red no bloqueen el hilo del Hub si un cliente web tiene lag momentáneo.
		outbound: make(chan []byte, 256),
	}
}

// Write corre en su propia Goroutine dedicada para cada cliente.
// Escucha el canal interno y escribe de forma secuencial hacia el socket físico.
func (c *Client) Write() {
	// MEJORA: Si la goroutine termina por cualquier motivo, nos aseguramos de limpiar
	// las conexiones físicas de red y remover al cliente del mapa del Hub.
	defer func() {
		c.Close()
	}()

	for {
		select {
		// Escucha si hay nuevos bytes listos para ser enviados al navegador
		case message, ok := <-c.outbound:
			if !ok {
				// Si el canal outbound se cerró de forma ordenada, notificamos al protocolo
				// WebSocket del cliente que la transmisión ha terminado.
				c.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Escribe los bytes puros codificados (ej. JSON del feed creado) como texto plano por la red
			if err := c.socket.WriteMessage(websocket.TextMessage, message); err != nil {
				// Si la escritura falla (ej: el cliente cerró la pestaña del navegador),
				// salimos del bucle para que el 'defer' limpie los recursos.
				return
			}
		}
	}
}

// Close clausura la conexión de red de forma segura y destruye los canales concurrentes.
// CORRECCIÓN: Cambiado de receptor por valor (c Client) a receptor por puntero (*Client)
func (c *Client) Close() {
	// once.Do garantiza bajo hilos concurrentes que el bloque de código interno
	// se ejecutará estrictamente una única vez, evitando el pánico "close of closed channel".
	c.once.Do(func() {
		// 1. Cierra la conexión física TCP/WebSocket
		if c.socket != nil {
			c.socket.Close()
		}
		// 2. Libera el canal de Go. Al cerrarse, romperá el bucle del método Write() automáticamente.
		close(c.outbound)

		// 3. Notificamos al Hub (si tu implementación tiene el método) para desregistrar al cliente.
		// c.hub.unregister <- c
	})
}

/*
El struct Client representa una conexión WebSocket persistente abierta con un navegador o cliente frontend, encargada de enviarle notificaciones asíncronas en cuanto ocurren eventos en tu arquitectura CQRS.
*/
