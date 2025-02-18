# Lightweight Golang Kite Library for Cash and FnO Analysis

This project provides a comprehensive solution for analyzing Cash and F&O (Futures & Options) data from Zerodha Kite. It features:

- Real-time data collection from Kite Trading API
- Data storage in ClickHouse database for high-performance analytics
- Grafana dashboards for data visualization and monitoring
- NATS message broker for reliable data streaming
- Telegram notifications for important updates
- Containerized with Docker for easy deployment and scaling

The entire stack runs in Docker containers, making it easy to set up and manage the complete infrastructure including:

- Kite data collector
- ClickHouse database
- Grafana visualization platform
- NATS message broker

## Environment Variables

To configure the application, rename `.env_example` to `.env` and set the following environment variables:

### Kite Trading API Configuration

- `TA_KITE_LOGINTYPE`: Authentication method for Kite (set to "WEB" for default authentication, and "API" for API-based authentication)
- `TA_KITE_ID`: Your Kite user ID/client ID
- `TA_KITE_PASSWORD`: Your Kite account password
- `TA_KITE_TOTP`: Time-based One-Time Password for 2FA authentication
- `TA_KITE_APIKEY`: API key from your Kite developer account ( required if TA_KITE_LOGINTYPE is API )
- `TA_KITE_APISECRET`: API secret from your Kite developer account ( required if TA_KITE_LOGINTYPE is API )
- `TA_KITE_PATH`: Path for Kite callback URL ( required if TA_KITE_LOGINTYPE is API )

### Digital Ocean Storage Configuration

- `TA_DO_KEY`: Digital Ocean Spaces access key
- `TA_DO_SECRET`: Digital Ocean Spaces secret key
- `TA_DO_BUCKET`: Digital Ocean Spaces bucket name
- `TA_DO_ENDPOINT`: Digital Ocean Spaces endpoint URL
- `TA_DO_REGION`: Digital Ocean Spaces region

### Notification Configuration

- `TA_TELEGRAM_TOKEN`: Telegram bot token for notifications
- `TA_TELEGRAM_ID`: Telegram chat ID for receiving notifications

### System Configuration

- `TZ`: Timezone setting (default: Asia/Kolkata)
- `DB_NAME`: Database name (default: default)
- `TA_NATS_URI`: NATS message broker URI (default: nats://nats:4222)
- `DB_URI`: ClickHouse database URI (default: clickhouse:9000)

```env
TA_KITE_LOGINTYPE=WEB
TA_KITE_ID=
TA_KITE_PASSWORD=
TA_KITE_TOTP=
TA_KITE_APIKEY=
TA_KITE_APISECRET=
TA_KITE_PATH=
TA_DO_KEY=
TA_DO_SECRET=
TA_DO_BUCKET=
TA_DO_ENDPOINT=
TA_DO_REGION=
TA_TELEGRAM_TOKEN=
TA_TELEGRAM_ID=
TZ=Asia/Kolkata
DB_NAME=default
TA_NATS_URI=nats://nats:4222
DB_URI=clickhouse:9000
```

## Running the Application

To run the application using Docker:

```bash
docker-compose up -d
```

Note: Make sure all environment variables are properly configured in your `.env` file before starting the application.
