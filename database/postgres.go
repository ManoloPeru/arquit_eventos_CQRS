package database

import (
	"context"
	"database/sql"

	// El guion bajo (_) realiza una "importación de efectos secundarios" (side-effect import).
	// Registra el driver de PostgreSQL ("postgres") dentro del paquete genérico database/sql
	// al ejecutar su función interna init(), permitiendo usar sql.Open("postgres", ...).
	_ "github.com/lib/pq"

	"platzi.com/go/cqrs/models"
)

// PostgresRepository implementa el patrón de diseño "Repository".
// Encapsula la conexión física a la base de datos para que la lógica de negocio (CQRS)
// no tenga que saber nada sobre consultas SQL ni drivers específicos.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository actúa como constructor (Factory Method).
// Ojo: sql.Open NO abre una conexión inmediata a la base de datos, solo valida que los
// argumentos del string de conexión (url) sean sintácticamente correctos y prepara el pool.
func NewPostgresRepository(url string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	// Tip pro: Si quisieras asegurar que la BD está realmente arriba y responde aquí,
	// se suele agregar un `if err := db.Ping(); err != nil { return nil, err }`

	return &PostgresRepository{db}, nil
}

// Close libera los recursos y cierra todas las conexiones abiertas en el Connection Pool.
// Se invoca típicamente mediante un `defer repo.Close()` en el punto de entrada (main.go).
func (repo *PostgresRepository) Close() {
	repo.db.Close()
}

// InsertFeed almacena un nuevo elemento en la tabla de lectura/escritura de feeds.
// Nota técnica sobre los placeholders: Al usar el driver 'pq', se utiliza la sintaxis
// posicional de Postgres ($1, $2, $3) en lugar de los signos de interrogación (?) de MySQL.
func (repo *PostgresRepository) InsertFeed(ctx context.Context, feed *models.Feed) error {
	// ExecContext es ideal para INSERT, UPDATE o DELETE ya que ejecuta la query
	// sin esperar un conjunto de filas (Rows) de retorno. El ctx maneja cancelaciones y timeouts.
	_, err := repo.db.ExecContext(
		ctx,
		"INSERT INTO feeds (id, title, description) VALUES ($1, $2, $3)",
		feed.ID,
		feed.Title,
		feed.Description,
	)
	return err
}

// ListFeeds recupera todos los registros de la tabla de lectura.
// En la arquitectura CQRS, este método representa la optimización del "Read Model",
// devolviendo proyecciones planas de los datos listos para el consumo de la API.
func (repo *PostgresRepository) ListFeeds(ctx context.Context) ([]*models.Feed, error) {
	// QueryContext se usa para consultas SELECT que esperan múltiples filas de vuelta.
	rows, err := repo.db.QueryContext(ctx, "SELECT id, title, description, created_at FROM feeds")
	if err != nil {
		return nil, err
	}
	// CRÍTICO: defer rows.Close() asegura que los cursores y conexiones del pool
	// asociados a esta consulta se liberen, previniendo fugas de memoria (memory leaks).
	defer rows.Close()

	feeds := []*models.Feed{}

	// rows.Next() itera fila por fila a través del cursor devuelto por Postgres (bucle streaming).
	for rows.Next() {
		feed := &models.Feed{}
		// Scan mapea el orden exacto de las columnas del SELECT a los punteros de la estructura.
		if err := rows.Scan(&feed.ID, &feed.Title, &feed.Description, &feed.CreatedAt); err != nil {
			return nil, err
		}
		// Agrega cada puntero de Feed al slice de resultados.
		feeds = append(feeds, feed)
	}

	// Es buena práctica verificar rows.Err() después de terminar el bucle para asegurarnos
	// de que el recorrido no se cortó abruptamente por un error de red a mitad del stream.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return feeds, nil
}
