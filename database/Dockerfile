# -----------------------------------------------------------------------------
# NOTA: La línea original 'FROM postgres:16-alpine' está comentada (no se ejecuta).
# Actualmente estás cambiando a una versión de Postgres mucho más antigua (10.3).
# Si estás usando funciones modernas de SQL, se recomienda mantener la versión 16.
# se comento: FROM postgres:10.3
# -----------------------------------------------------------------------------
FROM postgres:16-alpine

# Copia tu script de inicialización 'up.sql' al directorio especial de Postgres.
# Los archivos (.sql o .sh) dentro de '/docker-entrypoint-initdb.d/' se ejecutan
# automáticamente en orden alfabético la PRIMERA VEZ que el contenedor se levanta.
# Al renombrarlo como '1.sql', te aseguras de que sea el primer script en correr.
COPY up.sql /docker-entrypoint-initdb.d/1.sql

# Comando por defecto que arranca el motor de la base de datos PostgreSQL
# y lo mantiene escuchando peticiones en el puerto 5432.
CMD ["postgres"]