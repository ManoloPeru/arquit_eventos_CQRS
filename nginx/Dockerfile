# Usamos una versión estable, moderna y ultra ligera (Alpine Linux)
FROM nginx:1.27-alpine

# Limpiamos archivos de configuración por defecto de Nginx para evitar conflictos de ruteo
RUN rm -rf /etc/nginx/conf.d/*

# Copiamos tu archivo de configuración optimizado (con los upstream hacia feed, query y pusher)
COPY nginx.conf /etc/nginx/nginx.conf

# Exponemos explícitamente el puerto 80 (buena práctica de documentación para Docker Compose)
EXPOSE 80

# Mantenemos el comando nativo para ejecutar en primer plano de forma estable
CMD ["nginx", "-g", "daemon off;"]