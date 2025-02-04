# AI Playground

Welcome to the **AI Playground** repository. This is a simple, open source project providing a real-time chat experience with a Go backend and a frontend built using Svelte, TypeScript, and Vite.

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](#)
[![License](https://img.shields.io/badge/license-MIT-blue)](#)
[![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-orange)](#)

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Technology Stack](#technology-stack)
- [Repository Structure](#repository-structure)
- [Getting Started](#getting-started)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

This repository features a straightforward AI Playground, created as a hobby project to explore real-time chat functionality using modern web technologies.

---

## Features

- **Real-time Chat**: Basic chat features with immediate updates.
- **Responsive Interface**: Usable on various devices.
- **Modern Technologies**: Built with Svelte, TypeScript, Vite, and Go.

---

## Technology Stack

- **Frontend**: Developed with Svelte, TypeScript, and Vite.
- **Backend**: Developed in Go, focusing on simplicity and performance.

---

## Repository Structure

- **frontend/**: Contains the Svelte + TypeScript + Vite application.
- **backend/**: Contains the Go backend service.

---

## Getting Started

1. **Clone the repository:**
    ```bash
    git clone https://github.com/thomas-cabral/ai-playground
    ```
2. **Setup the Frontend:**
    ```bash
    cd frontend
    npm install
    npm run dev
    ```
3. **Setup the Backend:**
    ```bash
    cd backend
    go run main.go
    ```

4. **Optional Development Setup (Backend):**
    For hot-reloading during development, you can use [Air](https://github.com/cosmtrek/air):
    ```bash
    # Install Air install 
    go install github.com/air-verse/air@latest

    # Run the backend with Air
    cd backend
    air
    ```
    This will automatically rebuild and restart the server when you make changes.


---

*Thank you for checking out this project.*