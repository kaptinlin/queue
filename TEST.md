# Testing Guide

## Available Make Targets

We provide several Make targets for testing convenience:

```bash
# View all available commands
make help
```

## Running Tests

### Quick Method (Recommended)

Use the automated test target that handles Redis setup and cleanup:

```bash
# Start Redis, run tests, then cleanup automatically
make test-with-redis
```

### Manual Method

For more control over the testing process:

```bash
# Start Redis service
make redis

# Run all tests (requires Redis to be running)
make test

# Stop Redis service when done
make redis-stop
```

### Direct Go Test Command

You can also run tests directly if Redis is already running:

```bash
cd tests && go test -v
```

## Managing Redis Service

### Starting Redis

```bash
# Start Redis service and wait for it to be ready
make redis
```

### Stopping Redis

```bash
# Stop and remove Redis containers
make redis-stop

# Or use docker-compose directly for advanced options
docker-compose down -v  # Also removes data volumes
```

## Viewing Redis Logs

```bash
# View Redis logs
docker-compose logs redis

# Follow logs in real-time
docker-compose logs -f redis
```

## Connecting to Redis

```bash
# Connect using redis-cli
docker-compose exec redis redis-cli

# Or use local redis-cli (if installed)
redis-cli -h localhost -p 6379
``` 
