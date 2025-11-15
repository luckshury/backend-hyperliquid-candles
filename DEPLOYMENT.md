# Quick Deployment Guide

## ✅ Project Status

Your Hyperliquid backend server is **ready to deploy**! All code has been created and tested.

## 📁 Files Created

```
backend for hyperliquid/
├── main.go              # HTTP server with CORS, gzip, logging
├── worker.go            # CandleFetcherActor - background worker
├── symbols.go           # SymbolFetcherActor - symbol discovery
├── cache.go             # Thread-safe in-memory cache
├── hyperliquid.go       # Hyperliquid API client
├── hydromancer.go       # Symbol fetching via Hyperliquid meta API
├── types.go             # Data structures
├── go.mod               # Go dependencies
├── go.sum               # Dependency checksums (auto-generated)
├── Dockerfile           # Multi-stage Docker build
├── railway.json         # Railway deployment config
├── env.example          # Environment variables template
├── .gitignore           # Git ignore rules
└── README.md            # Full documentation
```

## 🚀 Local Testing

The server has been tested and is working correctly:

```bash
# Health check response:
{
  "status": "healthy",
  "symbol_count": 184,
  "last_update": "2025-11-15T18:10:45Z",
  "symbol_update": "2025-11-15T18:10:07Z"
}

# Symbols endpoint shows 184 active perpetual pairs:
["BTC", "ETH", "ATOM", "DYDX", "SOL", "AVAX", "BNB", ...]

# BTC candles endpoint returns 169 hourly candles (7 days):
{
  "symbol": "BTC",
  "candles": [
    {
      "timestamp": 1762628399999,
      "open": 101953,
      "high": 102200,
      "low": 101796,
      "close": 102111,
      "volume": 722.40228
    },
    ...
  ],
  "last_update": "2025-11-15T18:10:45Z"
}
```

## 🏃 Run Locally

```bash
cd "/Users/kayadacosta/backend for hyperliquid"

# Build
go build -o server .

# Run
./server

# Server will start on port 3000
# Visit: http://localhost:3000/health
```

## ☁️ Deploy to Railway

### Option 1: Via Railway CLI (Recommended)

```bash
# Install Railway CLI
npm install -g @railway/cli

# Login
railway login

# Initialize project
railway init

# Set environment variables
railway variables set HYDROMANCER_API_KEY=sk_nNhuLkdGdW5sxnYec33C2FBPzLjXBnEd
railway variables set PORT=3000
railway variables set CANDLE_INTERVAL=1h
railway variables set CANDLE_DAYS=7
railway variables set REFRESH_INTERVAL_MIN=5
railway variables set SYMBOL_REFRESH_INTERVAL_MIN=60

# Deploy!
railway up

# Get your public URL
railway domain
```

### Option 2: Via Railway Dashboard

1. Go to [Railway.app](https://railway.app)
2. Click "New Project" → "Deploy from GitHub repo"
3. Connect your GitHub account and select this repo
4. Railway will auto-detect the Dockerfile
5. Add environment variables:
   - `HYDROMANCER_API_KEY`: `sk_nNhuLkdGdW5sxnYec33C2FBPzLjXBnEd`
   - `PORT`: `3000`
   - `CANDLE_INTERVAL`: `1h`
   - `CANDLE_DAYS`: `7`
   - `REFRESH_INTERVAL_MIN`: `5`
   - `SYMBOL_REFRESH_INTERVAL_MIN`: `60`
6. Click "Deploy"
7. Once deployed, go to Settings → Generate Domain

## 📊 What the Server Does

1. **On Startup:**
   - Fetches all 184 active Hyperliquid perpetual symbols
   - Fetches 7 days of 1h candle data for each symbol
   - Stores everything in memory

2. **Background Workers (Hollywood Actors):**
   - **SymbolFetcherActor**: Updates symbol list every 60 minutes
   - **CandleFetcherActor**: Refreshes candle data every 5 minutes

3. **API Endpoints:**
   - `GET /health` - Health check with stats
   - `GET /api/symbols` - List all symbols
   - `GET /api/candles` - All candle data (gzipped)
   - `GET /api/candles/:symbol` - Candles for specific symbol

4. **Features:**
   - ✅ CORS enabled for all origins
   - ✅ Gzip compression
   - ✅ ETag caching
   - ✅ Request logging
   - ✅ Graceful shutdown
   - ✅ Rate limit handling with retries
   - ✅ Thread-safe cache

## 🔧 Configuration

All configuration is done via environment variables (see `env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | Server port |
| `HYDROMANCER_API_KEY` | Required | API key (currently not used, but kept for future) |
| `CANDLE_INTERVAL` | `1h` | Candle timeframe (1m, 5m, 15m, 1h, 4h, 1d) |
| `CANDLE_DAYS` | `7` | Days of historical data |
| `REFRESH_INTERVAL_MIN` | `5` | Candle refresh interval (minutes) |
| `SYMBOL_REFRESH_INTERVAL_MIN` | `60` | Symbol list refresh (minutes) |

## 🎯 Architecture Highlights

### Hollywood Actor Pattern

This project uses the **Hollywood** framework for Go, which provides:

- **Concurrent actors** that process messages independently
- **Automatic scheduling** of recurring tasks
- **Fault isolation** - one actor failing doesn't crash the system
- **Message passing** instead of shared state

The two main actors are:

1. **SymbolFetcherActor** (`symbols.go`)
   - Discovers symbols via Hyperliquid's `meta` API
   - Runs every 60 minutes
   - Caches symbols as fallback

2. **CandleFetcherActor** (`worker.go`)
   - Fetches candles for all symbols
   - Processes in batches of 10 (concurrent)
   - 200ms delay between batches (rate limit friendly)
   - Runs every 5 minutes

### Thread-Safe Cache

The `Cache` struct (`cache.go`) uses `sync.RWMutex` to safely handle:
- Multiple concurrent readers
- Exclusive writes
- No race conditions

## ⚠️ Known Behaviors

### Rate Limiting (HTTP 429)

Hyperliquid's API has rate limits. When fetching 184 symbols:
- Some requests may fail with 429 errors
- Failed symbols get empty candle arrays
- They'll be retried on the next cycle (5 minutes)
- This is expected and handled gracefully

Typical success rate: ~80% (147/184 symbols on first fetch)

### Memory Usage

- ~50-100MB for 184 symbols × 169 candles each
- All data stored in memory (fast access)
- No database required

### Startup Time

- Initial symbol fetch: ~1 second
- Initial candle fetch: ~40 seconds
- Total startup: ~45 seconds

## 🧪 Testing the Deployment

After deploying to Railway, test your endpoints:

```bash
# Replace YOUR-APP.railway.app with your actual domain
BASE_URL="https://YOUR-APP.railway.app"

# Health check
curl $BASE_URL/health

# Get all symbols
curl $BASE_URL/api/symbols

# Get BTC candles
curl $BASE_URL/api/candles/BTC

# Get all candles (compressed)
curl -H "Accept-Encoding: gzip" $BASE_URL/api/candles | gunzip | head -c 1000
```

## 📝 Next Steps

1. **Push to GitHub:**
   ```bash
   git init
   git add .
   git commit -m "Initial commit - Hyperliquid candle backend"
   git remote add origin YOUR-REPO-URL
   git push -u origin main
   ```

2. **Deploy to Railway** using one of the methods above

3. **Monitor the logs** in Railway dashboard to see:
   - Symbol fetching
   - Candle batch processing
   - API requests

4. **Optional improvements:**
   - Add PostgreSQL for persistent storage
   - Add Redis for distributed caching
   - Add Prometheus metrics
   - Add rate limiting middleware
   - Add authentication

## 🐛 Troubleshooting

**Server won't start:**
- Check PORT is not already in use
- Verify Go 1.21+ is installed: `go version`

**No symbols found:**
- Check internet connection
- Hyperliquid API might be down (check their status)

**High memory usage:**
- Reduce `CANDLE_DAYS` from 7 to 3 or 1
- Reduce `CANDLE_INTERVAL` to smaller timeframe

**Rate limit errors:**
- Increase batch delay in `worker.go`
- Reduce concurrent batch size from 10 to 5

## 📚 Documentation

See `README.md` for full documentation including:
- Complete API reference
- Architecture details
- Performance metrics
- Error handling
- Contributing guidelines

---

**Built with ❤️ using Hollywood Actor Framework**

