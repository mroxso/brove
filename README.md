# BROVE Relay

A private nostr relay implementation built with Go, designed for secure and controlled access to decentralized messaging on the nostr protocol.

## Overview

This is a custom nostr relay server that provides a private, authenticated relay service for the nostr protocol. The relay implements a whitelist-based access control system where only authorized public keys can read from and write to the relay.

## Features

- **Private Access Control**: Only whitelisted public keys can read from or write to the relay
- **NIP-86 Management API**: Full relay management capabilities including user management
- **PostgreSQL Backend**: Reliable event storage using PostgreSQL database
- **Docker Support**: Easy deployment with Docker Compose
- **Web Interface**: Basic web interface served at the root endpoint
- **Authentication Required**: AUTH message support for secure access
- **Owner Privileges**: Relay owner has full access and management capabilities

## Architecture

The relay is built using:
- **[Khatru](https://github.com/fiatjaf/khatru)**: nostr relay framework
- **PostgreSQL**: Event storage and user management
- **Go**: Backend implementation
- **Docker**: Containerized deployment

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/mroxso/brove.git
cd brove
```

2. Configure environment variables in `compose.yaml`:
```yaml
environment:
  - RELAY_NAME=Your Relay Name
  - RELAY_PUBKEY=your_relay_owner_pubkey_here
  - RELAY_DESCRIPTION=Your relay description
  - RELAY_ICON=https://your-icon-url.com/icon.jpg
```

3. Start the relay:
```bash
docker compose up -d
```

The relay will be available at:
- **Relay**: `ws://localhost:3334`
- **Web Interface**: `http://localhost:3334`
- **pgAdmin**: `http://localhost:15432` (admin: user@example.me, password: nostr#pgadmin)

### Manual Installation

1. Install dependencies:
```bash
go mod download
```

2. Set up PostgreSQL database and update connection string in `main.go`

3. Set environment variables:
```bash
export RELAY_NAME="Your Relay Name"
export RELAY_PUBKEY="your_relay_owner_pubkey_here"
export RELAY_DESCRIPTION="Your relay description"
export RELAY_ICON="https://your-icon-url.com/icon.jpg"
```

4. Run the relay:
```bash
go run .
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RELAY_NAME` | Name of the relay | "brove relay" |
| `RELAY_PUBKEY` | Owner's public key (hex format) | "82c1b69ddb84fb9a8cc68616118a9a1c794dfeb29c8d2ea2cec59af21f9df804" |
| `RELAY_DESCRIPTION` | Relay description | "this is my custom and private relay" |
| `RELAY_ICON` | URL to relay icon | Default probe image |

### Database Configuration

The relay uses PostgreSQL for both event storage and user management. Connection details are configured in the code:

```go
"postgresql://postgres:postgres@db:5432/khatru-relay?sslmode=disable"
```

## Access Control

### Reading Events
- Requires authentication via AUTH message
- Only whitelisted public keys can read events
- Relay owner always has read access

### Writing Events
- Only whitelisted public keys can write events
- Relay owner always has write access
- Events are validated for proper format and signatures

### Management API (NIP-86)

The relay implements NIP-86 management endpoints for:
- Adding allowed public keys
- Removing public keys from allowlist
- Listing allowed public keys
- Relay owner authentication required

## API Endpoints

- `ws://localhost:3334` - WebSocket NOSTR relay endpoint
- `http://localhost:3334` - Web interface
- `http://localhost:3334/.well-known/nostr/management` - NIP-86 management API

## User Management

### Adding Users

Only the relay owner can manage users. Use a NIP-86 compatible client or direct API calls to:

1. Authenticate as the relay owner
2. Call the management API to add public keys to the allowlist

### Database Schema

The relay maintains an `allowed_pubkeys` table:

```sql
CREATE TABLE allowed_pubkeys (
    pubkey VARCHAR(64) PRIMARY KEY,
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Development

### Building

```bash
go build -o relay .
```

### Running Tests

```bash
go test ./...
```

### Docker Build

```bash
docker build -t brove .
```

## Security Considerations

- Change the default relay owner public key
- Use secure PostgreSQL credentials
- Consider using SSL/TLS in production
- Regularly backup the database
- Monitor relay access logs

## NOSTR Implementation Details

### Supported NIPs
- NIP-01: Basic protocol flow
- NIP-11: Relay information document
- NIP-86: Relay management API
- Authentication and access control

### Event Policies
- Valid event kind validation
- Large tag prevention (max 100 tags)
- Public key authorization checking
- Event signature validation

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source. Please check the license file for details.

## Support

For issues and questions:
- Create an issue on GitHub
- Contact the maintainer: @highperfocused

## Acknowledgments

- Built with [Khatru](https://github.com/fiatjaf/khatru) nostr relay framework
- Thanks to the nostr development community
- PostgreSQL event store implementation by fiatjaf