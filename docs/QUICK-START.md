# Quick Start Guide

In this guide, we will get the InfraSense platform running on a fresh Ubuntu server using Docker.

## Prerequisites

- Ubuntu 24.04 server (or another modern Linux distribution)
- Root or `sudo` access
- At least 4GB of RAM (8GB recommended for production)

## Installation Steps

1.  **Clone the Repository**

    ```bash
    git clone https://github.com/tanjil-al-mahmud/infrasense.git
    cd infrasense
    ```

2.  **Configure Environment**

    Copy the example environment file and create your configuration.

    ```bash
    cp .env.example .env
    ```

    *Optional:* Edit `.env` to change default passwords, database names, or API keys.

3.  **Start the Services**

    We use Docker Compose to manage all dependencies (PostgreSQL, VictoriaMetrics, Grafana, Loki, Prometheus, AlertManager, Backend, Frontend).

    ```bash
    cd deploy
    docker compose --env-file ../.env up -d --build
    ```

    This command will download all necessary images, build custom images locally, and start the containers in the background.

4.  **Verify the Services**

    Ensure all containers are running and healthy.

    ```bash
    docker compose ps
    ```

5.  **Access the Dashboard**

    Open your web browser and go to `http://<your-server-ip>`.

    Log in using the credentials defined in your `.env` file (default: `admin` / `admin123`).

## Next Steps

- Proceed to [Adding Devices](ADDING_DEVICES.md) to start monitoring your hardware.
- Review [Configuration](CONFIGURATION.md) to customize alerts, notifications, and retention policies.
