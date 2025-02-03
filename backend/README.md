
# Backend Service

This is the backend service for the AI chat application.

## Setup

1. Make sure you have Go installed on your system
2. Clone the repository
3. Navigate to the backend directory
4. Install dependencies:

```bash
go mod tidy
```


You can get an OpenRouter API key by:
1. Going to [OpenRouter](https://openrouter.ai/)
2. Creating an account
3. Generating an API key from your dashboard


## Environment Variables

Create a `.env` file in the backend directory with the following variables:

```
OPENROUTER_API_KEY=<your-openrouter-api-key>
```

## Running the Service

To run the backend service:

```bash
go run main.go
```

## Running the Service with Air

Air is a tool that allows you to run the service and automatically reload it on file changes.

```bash
air
```



