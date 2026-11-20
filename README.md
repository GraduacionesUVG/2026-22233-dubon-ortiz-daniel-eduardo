# Gimpy: API de planificación de rutinas personalizadas mediante inteligencia artificial adaptativa

## Descripción

El abandono temprano en los gimnasios es frecuente, y una de sus causas es que las rutinas son fijas y no se adaptan al estado del usuario en cada día. **Gimpy** es una API RESTful que genera rutinas de entrenamiento personalizadas a partir de lo que el usuario declara en una conversación en lenguaje natural: su nivel de energía, el tiempo disponible, sus lesiones y sus objetivos.

Un agente basado en Claude (Anthropic) con *tool calling* consulta el historial de entrenamientos, la fatiga y las lesiones del usuario, y apoyándose en un catálogo controlado de ejercicios arma o ajusta la rutina del día. La respuesta se entrega en tiempo real por *streaming* (Server-Sent Events). Además, el agente:

- atiende únicamente en español y rechaza otros idiomas;
- limita la nutrición a consejos generales de apoyo al entrenamiento (sin dietas, calorías ni dosis de suplementos);
- registra la retroalimentación del usuario (fatiga y notas) como un diario de entrenamiento;
- adjunta videos de referencia de cada ejercicio.

**Alcance de esta entrega:** este repositorio contiene únicamente el **backend** (la API). El cliente web (frontend) está fuera del alcance de este trabajo de graduación y su código **no se incluye**. Las capturas y el video de la demo usan una app web de apoyo únicamente como herramienta para mostrar el backend en funcionamiento; la API en sí se prueba directamente por HTTP (`curl`), como se explica en la sección Instalación.

### Arquitectura

El proyecto sigue una arquitectura hexagonal (puertos y adaptadores):

| Capa | Ruta | Responsabilidad |
|------|------|-----------------|
| Dominio | `src/internal/domain/` | Entidades de negocio puras (usuario, rutina, ejercicio) |
| Puertos | `src/internal/ports/` | Interfaces (`UserRepository`, `WorkoutRepository`, `LLMService`, ...) |
| Casos de uso | `src/internal/usecases/` | Lógica de negocio: autenticación y generación de rutinas |
| Adaptadores | `src/internal/adapters/` | PostgreSQL, cliente de Anthropic y búsqueda de videos de YouTube |
| Entrega HTTP | `src/internal/delivery/http/` | Handlers, autenticación JWT, CORS y límite de solicitudes |
| Entrada | `src/cmd/api/` | `main.go`: configuración e inyección de dependencias |

### Endpoints

| Método | Ruta | Autenticación | Descripción |
|--------|------|---------------|-------------|
| GET | `/health` | No | Verificación de estado del servicio |
| POST | `/auth/register` | No | Registro de usuario |
| POST | `/auth/login` | No | Inicio de sesión (devuelve un JWT) |
| POST | `/routine` | JWT | Genera una rutina o responde como entrenador, por SSE |
| GET | `/routine/history` | JWT | Historial de rutinas del usuario |

## Tecnologías Utilizadas

- Go 1.26 (biblioteca estándar `net/http`, sin frameworks web)
- PostgreSQL 16 (`pgx`)
- Claude de Anthropic con *tool calling* (`anthropic-sdk-go`), modelo `claude-sonnet-4-6`
- JWT (`golang-jwt`) y `bcrypt` (`golang.org/x/crypto`) para autenticación
- Server-Sent Events (SSE) para el *streaming* de respuestas
- YouTube Data API v3 (opcional) para los videos de los ejercicios
- Docker y Testcontainers (pruebas de integración) y k6 (pruebas de carga)

## Requisitos Previos

- [Go](https://go.dev/dl/) 1.26 o superior
- [Docker](https://docs.docker.com/get-docker/) (para PostgreSQL 16 y para las pruebas de integración con Testcontainers)
- Cliente `psql` de PostgreSQL (opcional: también se pueden ejecutar los scripts con `docker exec`, como se indica abajo)
- `curl` para probar la API
- Una API key de Anthropic ([console.anthropic.com](https://console.anthropic.com/))
- [k6](https://k6.io/docs/get-started/installation/) (opcional, solo para las pruebas de carga)

## Instalación

1. Clonar el repositorio:
   ```bash
   git clone https://github.com/DanielDubon/2026-22233-dubon-ortiz-daniel-eduardo.git
   cd 2026-22233-dubon-ortiz-daniel-eduardo/src
   ```
   Los comandos siguientes se ejecutan desde la carpeta `src/`.

2. Levantar PostgreSQL y crear el esquema de la base de datos:
   ```bash
   docker run -d --name gimpy-db -e POSTGRES_PASSWORD=gimpy -e POSTGRES_DB=gimpy \
     -p 127.0.0.1:5432:5432 postgres:16

   export DATABASE_URL='postgres://postgres:gimpy@localhost:5432/gimpy?sslmode=disable'
   psql "$DATABASE_URL" -f scripts/db/init.sql
   psql "$DATABASE_URL" -f scripts/db/create_chat_messages_table.sql
   psql "$DATABASE_URL" -f scripts/db/migrate_injuries_to_jsonb.sql
   psql "$DATABASE_URL" -f scripts/db/seed_exercises.sql
   ```
   Si no tienes `psql` instalado, ejecuta cada script con Docker, por ejemplo:
   ```bash
   docker exec -i gimpy-db psql -U postgres -d gimpy < scripts/db/init.sql
   ```
   El orden importa: `init.sql` crea las tablas, `create_chat_messages_table.sql` agrega el historial de conversación, `migrate_injuries_to_jsonb.sql` deja las lesiones en su formato estructurado y `seed_exercises.sql` carga el catálogo de 90 ejercicios.

3. Instalar las dependencias de Go:
   ```bash
   go mod download
   ```

4. Configurar las variables de entorno:
   ```bash
   cp .env.example .env
   # Editar .env: como mínimo ANTHROPIC_API_KEY y JWT_SECRET (por ejemplo, openssl rand -hex 32)
   ```

   | Variable | Requerida | Descripción |
   |----------|-----------|-------------|
   | `ANTHROPIC_API_KEY` | Sí | API key de Anthropic |
   | `DATABASE_URL` | Sí | Cadena de conexión a PostgreSQL |
   | `JWT_SECRET` | Sí | Secreto para firmar los JWT (sin valor por defecto) |
   | `ANTHROPIC_MODEL` | Sí, en la práctica | Modelo de Claude; usar `claude-sonnet-4-6` (el valor por defecto del código ya no existe en la API) |
   | `YOUTUBE_API_KEY` | No | Sin ella, los videos se resuelven como un enlace de búsqueda de YouTube |
   | `APP_TIMEZONE` | No | Zona horaria IANA (por defecto `America/Guatemala`) |
   | `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` | No | Límite de `POST /routine` por usuario (por defecto 3 por minuto) |

5. Ejecutar el proyecto (el servidor escucha en el puerto `8080`):
   ```bash
   go run ./cmd/api
   ```
   Verificar que responde:
   ```bash
   curl http://localhost:8080/health
   # {"status":"ok"}
   ```

### Probar la API

```bash
# 1. Registrar un usuario
curl -X POST http://localhost:8080/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"<contraseña>","name":"Usuario Demo","fitness_goals":["ganar masa muscular"],"injuries":[]}'

# 2. Iniciar sesión y guardar el token
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"<contraseña>"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

# 3. Pedir una rutina (respuesta en streaming SSE; -N desactiva el buffer de curl)
curl -N -X POST http://localhost:8080/routine \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"prompt":"Genera una rutina rápida de pierna para hipertrofia, tengo 30 minutos y energía 8 de 10"}'

# 4. Consultar el historial
curl http://localhost:8080/routine/history -H "Authorization: Bearer $TOKEN"
```

La respuesta de `POST /routine` es un flujo de eventos: múltiples líneas `data:` con fragmentos de texto (`{"type":"text","content":"..."}`) y, al terminar, un evento `event: complete` con la rutina estructurada (ejercicios, series y repeticiones). Si ocurre un error interno se emite `event: error` con un mensaje genérico.

### Pruebas

```bash
go build ./...
go test ./internal/...     # las pruebas de persistence requieren Docker (Testcontainers)
```

Las pruebas de carga con k6 están en `tests/stress/` y reciben la URL y las credenciales de un usuario ya registrado por variables de entorno:

```bash
k6 run -e BASE_URL=http://localhost:8080 tests/stress/load_test.js
k6 run -e BASE_URL=http://localhost:8080 -e TEST_EMAIL=demo@example.com -e TEST_PASSWORD=<contraseña> tests/stress/login_test.js
```

`POST /routine` está limitado por usuario; para `tests/stress/routine_test.js` inicia el servidor con `RATE_LIMIT_RPS=1000` y `RATE_LIMIT_BURST=1000`, de lo contrario verás respuestas `429`.

## Capturas

Las capturas de pantalla del sistema se encuentran en [`/capturas`](capturas/) (mínimo 10).

Como el cliente (frontend) está fuera del alcance de esta entrega y su código no se incluye, las capturas muestran una app web de apoyo, en viewport móvil, funcionando contra el backend real. Son 20 capturas de una sesión de prueba con usuarios ficticios y una base de datos local desechable (las contraseñas aparecen ocultas). Se ordenan según el flujo de uso:

| # | Archivo | Qué muestra |
|---|---------|-------------|
| 1 | [`captura-01.png`](capturas/captura-01.png) | Pantalla de inicio de sesión |
| 2 | [`captura-02.png`](capturas/captura-02.png) | Error de inicio de sesión con credenciales incorrectas (HTTP 401 del backend) |
| 3 | [`captura-03.png`](capturas/captura-03.png) | Registro de usuario con objetivo y lesión (contraseña oculta) |
| 4 | [`captura-04.png`](capturas/captura-04.png) | Chat con el coach tras el registro |
| 5 | [`captura-05.png`](capturas/captura-05.png) | Mensaje en lenguaje natural escrito para el coach |
| 6 | [`captura-06.png`](capturas/captura-06.png) | Respuesta del coach en streaming (SSE), a medio generar |
| 7 | [`captura-07.png`](capturas/captura-07.png) | Respuesta completa con el botón para ver la rutina |
| 8 | [`captura-08.png`](capturas/captura-08.png) | Pestaña Rutina: rutina generada por el agente |
| 9 | [`captura-09.png`](capturas/captura-09.png) | Detalle de la rutina: ejercicios, series, repeticiones y peso |
| 10 | [`captura-10.png`](capturas/captura-10.png) | Detalle de la rutina: más ejercicios editables |
| 11 | [`captura-11.png`](capturas/captura-11.png) | Rutina guardada en “Mis Rutinas Guardadas” |
| 12 | [`captura-12.png`](capturas/captura-12.png) | Entrenamiento guiado: series completadas y descanso en curso |
| 13 | [`captura-13.png`](capturas/captura-13.png) | Entrenamiento completado: opción de reportar al coach |
| 14 | [`captura-14.png`](capturas/captura-14.png) | Reporte de resultados listo para enviar al coach (fatiga 7 de 10) |
| 15 | [`captura-15.png`](capturas/captura-15.png) | El coach registra la retroalimentación en el diario de entrenamiento |
| 16 | [`captura-16.png`](capturas/captura-16.png) | Adaptación al estado del día: cansancio y dolor de rodilla |
| 17 | [`captura-17.png`](capturas/captura-17.png) | Rutina adaptada: menos volumen y ejercicios acordes con su estado |
| 18 | [`captura-18.png`](capturas/captura-18.png) | Guardarraíl (usuario nuevo): el coach rechaza consultas fuera del dominio del fitness |
| 19 | [`captura-19.png`](capturas/captura-19.png) | Guardarraíl de idioma (usuario nuevo): el coach atiende únicamente en español |
| 20 | [`captura-20.png`](capturas/captura-20.png) | Límite de solicitudes: el backend responde HTTP 429 y la app lo informa |

## Demo

El video demostrativo se encuentra en [`/demo/demo.mp4`](demo/demo.mp4)

El video recorre el sistema completo con una app web de apoyo (cuyo código no forma parte de esta entrega): registro, generación de una rutina por *streaming*, entrenamiento guiado, retroalimentación al coach, adaptación al estado del día, guardarraíles del agente y límite de solicitudes. El video está narrado y, además, cada paso se explica con un subtítulo en pantalla.

## Documentación

El informe final del proyecto está disponible en [`/docs/informe_final.pdf`](docs/informe_final.pdf)

El consentimiento de uso de foto y video firmado se encuentra en [`/docs/consentimiento_firmado.pdf`](docs/consentimiento_firmado.pdf)

## Autores

- Daniel Eduardo Dubón Ortiz - 22233

Asesor: Ing. Douglas Barrios.

## Licencia

Copyright (c) 2026 Daniel Eduardo Dubón Ortiz. Todos los derechos reservados.

No se concede permiso de uso, copia, modificación ni distribución de este software sin autorización escrita del autor, salvo para revisar y evaluar este trabajo de graduación en la Universidad del Valle de Guatemala. Consulta el archivo [LICENSE](LICENSE) para los términos completos.
