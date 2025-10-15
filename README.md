# NearTalk - Technical Architecture Document

**NearTalk** is a proximity-based, anonymous chat and voice platform that allows people in physical proximity to connect and communicate in real-time without accounts or persistent data storage.

### Core Features
- Anonymous, no-signup access
- Distance-based discovery (100m - 2km radius)
- Group chat with auto-expiring messages (30 min TTL)
- Voice rooms with overlapping proximity circles
- Username changes (2-3 times limit)
- Privacy-preserving distance display (approximate, not exact location)

---

## 🏗️ System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Web App (React/Vue + Tailwind)                      │   │
│  │  - Geolocation API                                   │   │
│  │  - WebSocket Client (chat)                           │   │
│  │  - WebRTC Client (voice P2P)                         │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                      LOAD BALANCER                           │
│                    (nginx/Caddy/Traefik)                     │
└─────────────────────────────────────────────────────────────┘
                            │
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                     BACKEND LAYER (Go)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   HTTP API   │  │  WebSocket   │  │   Signaling  │     │
│  │   Server     │  │    Server    │  │    Server    │     │
│  │  (Gin/Fiber)│  │  (gorilla/ws)│  │   (WebRTC)   │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│                            │                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Core Services (Go packages)                   │  │
│  │  - Location Service (geohashing)                     │  │
│  │  - Session Manager (in-memory)                       │  │
│  │  - Message Router (proximity-based)                  │  │
│  │  - TTL Manager (cleanup goroutines)                  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    DATA LAYER (Minimal)                      │
│  ┌──────────────┐                    ┌──────────────┐      │
│  │    Redis     │                    │  PostgreSQL  │      │
│  │  (In-Memory) │                    │  (Minimal)   │      │
│  │  - Sessions  │                    │  - Rate Limit│      │
│  │  - Geohashes │                    │  - Analytics │      │
│  │  - Messages  │                    │              │      │
│  └──────────────┘                    └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```


**Signaling Flow**:
```
User A joins voice → Server finds overlapping users → 
Exchange SDP offers/answers → Establish P2P connection →
Audio streams directly between peers
```

**No audio routing through server** - pure P2P after signaling

**Geohashing Strategy**:
- Precision 7 geohash (~153m x 153m cells)
- Query neighboring cells for broader radius

**Features**:
- In-memory session storage (Redis fallback for multi-server)
- TTL-based expiration (30 min inactivity)
- Username change tracking (max 2-3 changes)






## 🎙️ Voice Connection Architecture

### WebRTC P2P Flow

```
User A                  Signaling Server               User B
  │                            │                          │
  │ 1. Join voice room         │                          │
  ├────────────────────────────>                          │
  │                            │                          │
  │ 2. Find overlapping users  │                          │
  │<────────────────────────────                          │
  │                            │                          │
  │ 3. Create offer (SDP)      │                          │
  ├────────────────────────────>                          │
  │                            │ 4. Forward offer         │
  │                            ├──────────────────────────>
  │                            │                          │
  │                            │ 5. Create answer (SDP)   │
  │                            │<─────────────────────────┤
  │ 6. Forward answer          │                          │
  │<────────────────────────────                          │
  │                            │                          │
  │ 7. Exchange ICE candidates │                          │
  │<────────────────────────────────────────────────────>│
  │                            │                          │
  │ 8. P2P connection established (audio streams)        │
  │<═══════════════════════════════════════════════════>│
  │            (no server involvement)                    │
```

## Deployment Architecture

```
┌──────────────────────────────────────┐
│  Cloudflare (CDN + DDoS Protection)  │
└──────────────────────────────────────┘
              │
              ↓
┌──────────────────────────────────────┐
│   nginx/Caddy (Reverse Proxy + TLS)  │
└──────────────────────────────────────┘
              │
              ↓
┌──────────────────────────────────────┐
│   Go Backend (Docker container)      │
│   - HTTP API: :8080                  │
│   - WebSocket: :8080/ws              │
│   - Signaling: :8080/signal          │
└──────────────────────────────────────┘
              │
              ↓
┌──────────────────────────────────────┐
│   Redis (Docker/Managed Service)     │
│   - Port: 6379                       │
│   - Persistence: Optional            │
└──────────────────────────────────────┘
```

## Development Phases

### Phase 1: MVP (2-3 weeks)
**Goal**: Basic chat working locally

- [ ] Frontend: Distance slider + chat UI
- [ ] Backend: HTTP API + WebSocket server
- [ ] Location service: Geohash-based proximity
- [ ] Redis integration: Sessions + messages
- [ ] Message TTL cleanup
- [ ] Deploy to single VPS

### Phase 2: Voice (1-2 weeks)
**Goal**: Add voice rooms

- [ ] WebRTC signaling server
- [ ] STUN/TURN integration
- [ ] Voice room UI
- [ ] Peer discovery

### Phase 3: Polish (1 week)
**Goal**: UX improvements

- [ ] Username change limits
- [ ] Better error handling
- [ ] Loading states
- [ ] Mobile responsive

---

## 📐 Mermaid Diagrams

### User Connection Flow
```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend
    participant A as API Server
    participant W as WebSocket
    participant R as Redis
    participant L as Location Service

    U->>F: Open site
    F->>U: Request geolocation permission
    U->>F: Grant permission
    F->>A: POST /api/session/create
    A->>R: Create session (30 min TTL)
    A->>F: Return sessionID
    F->>U: Show username prompt
    U->>F: Enter username
    F->>A: PATCH /api/session/username
    A->>R: Store username
    F->>W: Connect WebSocket (sessionID)
    W->>R: Store WS connection
    U->>F: Set distance radius (500m)
    F->>A: POST /api/location/update {lat, lon, radius}
    A->>L: UpdateLocation(sessionID, lat, lon, radius)
    L->>R: GEOADD + geohash index
    L->>W: Notify: Join geo-cell channel
    W->>R: SUBSCRIBE chat:{geohash}
    W->>F: Connected to chat
    F->>U: Show nearby users + chat
```

### Message Flow
```mermaid
sequenceDiagram
    participant UA as User A
    participant WA as WebSocket A
    participant R as Redis
    participant WB as WebSocket B
    participant UB as User B

    UA->>WA: Send message "Hi!"
    WA->>WA: Validate message
    WA->>R: PUBLISH chat:{geohash} {message}
    WA->>R: ZADD messages:{geohash} {timestamp} {message} EX 1800
    R->>WB: Broadcast to subscribers
    WB->>UB: Display message
    
    Note over R: After 30 minutes
    R->>R: Auto-delete message (TTL)
```

### Voice Connection Flow
```mermaid
sequenceDiagram
    participant UA as User A
    participant SA as Signaling
    participant UB as User B

    UA->>SA: Join voice room
    SA->>SA: Find overlapping users
    SA->>UA: Found User B (~200m away)
    UA->>SA: Create WebRTC offer (SDP)
    SA->>UB: Forward offer
    UB->>SA: Create answer (SDP)
    SA->>UA: Forward answer
    UA->>UB: Exchange ICE candidates
    Note over UA,UB: P2P connection established
    UA->>UB: Audio stream (direct)
    UB->>UA: Audio stream (direct)
```

### System Component Diagram
```mermaid
graph TB
    subgraph Client
        UI[Web UI<br/>React/Vue]
        WS_Client[WebSocket Client]
        WebRTC_Client[WebRTC Client]
        Geo[Geolocation API]
    end

    subgraph Backend
        LB[Load Balancer<br/>nginx]
        API[HTTP API Server<br/>Go/Gin]
        WS_Server[WebSocket Server<br/>gorilla/ws]
        Signal[Signaling Server<br/>WebRTC]
        
        subgraph Services
            LocSvc[Location Service<br/>Geohashing]
            SessionMgr[Session Manager]
            MsgRouter[Message Router]
            TTLMgr[TTL Manager]
        end
    end

    subgraph Data
        Redis[(Redis<br/>In-Memory)]
        PG[(PostgreSQL<br/>Analytics)]
    end

    UI --> LB
    WS_Client --> LB
    WebRTC_Client --> LB
    Geo --> UI

    LB --> API
    LB --> WS_Server
    LB --> Signal

    API --> LocSvc
    API --> SessionMgr
    WS_Server --> MsgRouter
    WS_Server --> TTLMgr

    LocSvc --> Redis
    SessionMgr --> Redis
    MsgRouter --> Redis
    TTLMgr --> Redis
    
    API --> PG
    TTLMgr --> PG

    classDef client fill:#e1f5ff,stroke:#01579b
    classDef backend fill:#fff3e0,stroke:#e65100
    classDef data fill:#f3e5f5,stroke:#4a148c
    
    class UI,WS_Client,WebRTC_Client,Geo client
    class LB,API,WS_Server,Signal,LocSvc,SessionMgr,MsgRouter,TTLMgr backend
    class Redis,PG data
```

### Geohash Proximity Diagram
```mermaid
graph TB
    subgraph User at dr5regw
        Center[Current Cell<br/>dr5regw]
    end

    subgraph Query Radius 500m
        N[North<br/>dr5regu]
        NE[NE<br/>dr5regv]
        E[East<br/>dr5regt]
        SE[SE<br/>dr5regs]
        S[South<br/>dr5regq]
        SW[SW<br/>dr5regm]
        W[West<br/>dr5regk]
        NW[NW<br/>dr5regp]
    end

    Center --> N
    Center --> NE
    Center --> E
    Center --> SE
    Center --> S
    Center --> SW
    Center --> W
    Center --> NW

    N --> Filter[Filter by actual distance]
    NE --> Filter
    E --> Filter
    SE --> Filter
    S --> Filter
    SW --> Filter
    W --> Filter
    NW --> Filter
    Center --> Filter

    Filter --> Results[Nearby Users<br/>~150m, ~200m, ~450m]
```
