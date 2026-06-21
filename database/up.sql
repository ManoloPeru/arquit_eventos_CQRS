-- -----------------------------------------------------------------------------
-- 1. LIMPIEZA PREVIA
-- Borra la tabla si ya existía para asegurar un despliegue limpio (Idempotencia).
-- Ojo: Cámbialo a "IF EXISTS feeds" (en plural) si quieres que coincida exactamente 
-- con las queries de tu repositorio en Go ("INSERT INTO feeds...").
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS feed;

-- -----------------------------------------------------------------------------
-- 2. CREACIÓN DE LA TABLA (Read Model / Proyección)
-- -----------------------------------------------------------------------------
CREATE TABLE feeds (
  -- ¡ALERTA AQUÍ, MANUEL! En tu archivo de Go estás usando la librería 'ksuid' o 
  -- strings autogenerados para el ID (ej: feed.ID). Si usas 'SERIAL', Postgres 
  -- esperará un entero autoincremental (1, 2, 3...) y tu Insert en Go va a fallar.
  -- 👉 RECOMENDACIÓN: Si usas KSUID o UUIDs en Go, cambia este tipo a: VARCHAR(27) o UUID
  id VARCHAR(27) PRIMARY KEY, --id SERIAL PRIMARY KEY,
  
  -- Título del feed. Restricción NOT NULL para asegurar la integridad de la data.
  title VARCHAR(255) NOT NULL,
  
  -- Descripción corta del feed. 
  description VARCHAR(255) NOT NULL,
  
  -- Fecha de creación del registro. Si tu aplicación de Go no envía este campo,
  -- Postgres ejecutará de forma nativa la función NOW() para asignarle la estampa de tiempo actual.
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

/*
Dado que en los pasos anteriores instalamos ksuid para manejar los identificadores únicos en tu bus de eventos, tu base de datos de lectura debería reflejar el mismo estándar. Si dejas el campo como SERIAL, chocará con los hashes de texto que genera KSUID.
La estructura ideal para que encaje perfectamente con tu código de Go sería esta:
CREATE TABLE feeds (
  id VARCHAR(27) PRIMARY KEY, -- Las KSUID de segmentio ocupan exactamente 27 caracteres de texto
  title VARCHAR(255) NOT NULL,
  description VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);