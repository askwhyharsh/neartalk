peoplearoundme/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
│
├── internal/                        # Private application code
│   ├── api/
│   │   ├── handler.go              # HTTP handlers
│   │   ├── middleware.go           # HTTP middleware (CORS, auth, etc.)
│   │   ├── routes.go               # Route definitions
│   │   └── response.go             # Standard API responses
│   │
│   ├── websocket/
│   │   ├── hub.go                  # WebSocket hub (manages connections)
│   │   ├── client.go               # WebSocket client connection
│   │   ├── handler.go              # WebSocket message handlers
│   │   └── message.go              # Message types and serialization
│   │
│   ├── location/
│   │   ├── service.go              # Location management
│   │   ├── geohash.go              # Geohash utilities
│   │   └── distance.go             # Distance calculations (Haversine)
│   │
│   ├── session/
│   │   ├── service.go              # Session management
│   │   ├── manager.go              # In-memory session store
│   │   └── username.go             # Username generation and validation
│   │
│   ├── ratelimit/
│   │   ├── limiter.go              # Rate limiter service
│   │   ├── middleware.go           # Rate limit middleware
│   │   └── config.go               # Rate limit configuration
│   │
│   ├── spam/
│   │   ├── detector.go             # Spam detection service
│   │   ├── profanity.go            # Profanity filter
│   │   └── patterns.go             # Spam pattern detection
│   │
│   ├── message/
│   │   ├── router.go               # Message routing logic
│   │   ├── store.go                # Message storage (Redis)
│   │   └── ttl.go                  # TTL manager
│   │
│   ├── storage/
│   │   ├── redis.go                # Redis client wrapper
│   │   └── postgres.go             # PostgreSQL client (optional)
│   │
│   └── config/
│       └── config.go               # Application configuration
│
├── pkg/                             # Public, reusable packages
│   ├── validator/
│   │   └── validator.go            # Input validation utilities
│   │
│   ├── logger/
│   │   └── logger.go               # Structured logging
│   │
│   └── errors/
│       └── errors.go               # Custom error types