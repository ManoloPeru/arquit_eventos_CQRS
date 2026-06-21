package repository

import (
	"context"

	"platzi.com/go/cqrs/models"
)

// Repository define el "contrato" o interfaz que cualquier motor de persistencia
// (Postgres, MySQL, MongoDB, o incluso un Mock en memoria para tests) debe cumplir.
// En CQRS, esto unifica las operaciones del Read y Write Model bajo un mismo comportamiento esperado.
type Repository interface {
	Close()                                                  // Libera el pool de conexiones.
	InsertFeed(ctx context.Context, feed *models.Feed) error // Abstracción para insertar datos.
	ListFeeds(ctx context.Context) ([]*models.Feed, error)   // Abstracción para consultar datos.
}

// 'repository' es una variable global interna (privada, empieza con minúscula)
// que almacenará la implementación concreta que hayamos elegido (por ejemplo, tu PostgresRepository).
var repository Repository

// SetRepository es una función de inyección de dependencias.
// Permite asignar la base de datos real al arrancar la aplicación en el main.go
// Ejemplo: repository.SetRepository(postgresRepo)
func SetRepository(repo Repository) {
	repository = repo
}

// Las siguientes funciones actúan como un patrón "Facade" (Fachada) o envoltorios (Wrappers).
// Permiten a otros paquetes llamar a los métodos del repositorio directamente usando
// 'repository.InsertFeed(...)' en lugar de tener que andar arrastrando la instancia del objeto.

// Close encapsula el cierre seguro del motor de persistencia activo.
func Close() {
	repository.Close()
}

// InsertFeed delega la inserción de datos a la implementación que esté guardada en la variable global.
func InsertFeed(ctx context.Context, feed *models.Feed) error {
	return repository.InsertFeed(ctx, feed)
}

// ListFeeds delega la consulta de la lista a la implementación concreta (ej: la query SELECT de Postgres).
func ListFeeds(ctx context.Context) ([]*models.Feed, error) {
	return repository.ListFeeds(ctx)
}

/*
Este patrón es fundamental en Go para lograr un desacoplamiento total entre la lógica de negocio y la base de datos (permitiéndote cambiar de Postgres a otra base de datos en el futuro sin tocar el resto del sistema)

si mañana quieres escribir pruebas unitarias (unit tests) para tu bus de eventos o lógica CQRS, no necesitas levantar Docker ni Postgres. Puedes crear un objeto simulado (Mock) en tu archivo de pruebas que implemente estos tres métodos y pasarlo usando SetRepository(mockRepo).
*/
