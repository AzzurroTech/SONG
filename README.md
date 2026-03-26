# SONG
**S**calable **O**pen **N**etwork **G**enerator  
*By Azzurro Technology Inc.*

SONG is a unified server-side framework that merges the concepts of **VENI** (Virtual Endpoints/Web Components), **VIDI** (Form-driven Database Management), and **VICI** (Context/Integration Management) into a single, dependency-free Go application. Built exclusively with the Go standard library, HTML, CSS, and vanilla JavaScript, SONG provides a lightweight, secure, and extensible platform for creating dynamic web interfaces and managing data without external bloat.

## Features

- **VENI Integration**: Create virtual endpoints that host dynamic HTML web components and scripts.
- **VIDI Engine**: Visual form builder that generates database schemas and handles data persistence via HTML forms.
- **VICI Context**: Centralized context management for session handling and cloud service integration.
- **Zero External Dependencies**: Runs entirely on the Go standard library (`net/http`, `html/template`, etc.).
- **PostgreSQL Ready**: Includes commented hooks and schema definitions for immediate PostgreSQL integration.
- **Dockerized**: Comes with a `Dockerfile` and `docker-compose.yml` for instant deployment with a database.

## Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose (optional, for full DB support)

### Local Development (Mock Mode)
1. Clone the repository:
   ```bash
   git clone https://github.com/azzurrotech/song.git
   cd song