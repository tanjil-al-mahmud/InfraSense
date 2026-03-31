# InfraSense Architecture

## Overview
InfraSense is a comprehensive monitoring and management platform designed specifically for managing diverse IT infrastructure environments. It employs a distributed, scalable microservices architecture that allows you to collect data from a wide variety of hardware devices globally.

## Core Components

### 1. Frontend (React)
- **Role:** The user interface for interacting with InfraSense.
- **Technology:** React, TypeScript, Vite.
- **Responsibilities:** Dashboard visualization, device management, user authentication, and system configuration.

### 2. Backend API Service
- **Role:** The central nervous system of the platform.
- **Technology:** Go (Golang), Fiber web framework.
- **Responsibilities:** Handling API requests from the frontend, managing connections to the database, orchestrating collectors, processing alerts, and acting as the central conduit for all data flow.

### 3. Collector Services
- **Role:** Specialized microservices that gather data from specific types of hardware protocols.
- **Technology:** Go (Golang).
- **Available Collectors:**
  - **SNMP Collector:** Gathers data from SNMP-enabled network switches, routers, and appliances.
  - **IPMI Collector:** Gathers hardware-level metrics (fan speed, temperature, power) from servers securely.
  - **Redfish Collector:** Interacts with modern REST APIs provided by server management interfaces.

### 4. Storage Layer
- **Relational Database (PostgreSQL):** Stores user data, system configuration, device configurations, and long-term settings.
- **Time-Series Database (VictoriaMetrics):** Highly optimized for storing and querying massive amounts of metric data collected from devices.
- **In-Memory Cache (Redis):** Caches frequent API queries, handles rate limiting, and manages active session states.

### 5. Notification Service
- **Role:** Manages the dissemination of system alerts and notifications.
- **Technology:** Go (Golang).
- **Responsibilities:** Delivering alerts triggered by metrics exceeding predefined thresholds via email, Slack, and other configured channels.

## Data Flow
1. **Discovery:** The user configures devices or triggers auto-discovery via the API.
2. **Collection:** The backend coordinates the appropriate collector microservices to poll the target devices.
3. **Ingestion:** Collectors push the gathered telemetries to the Time-Series DB (VictoriaMetrics).
4. **Alerting:** Dedicated alerting rules monitor the time-series data and forward incidents to the Notification Service when conditions are met.
5. **Visualization:** The Frontend queries the Backend API, which pulls combined data from PostgreSQL and VictoriaMetrics to present a cohesive real-time view to the user.
