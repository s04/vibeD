package builder

import "strings"

// DetectLanguage inspects the file map and returns the best-guess language.
// Returns "static", "nodejs", "python", or "go".
func DetectLanguage(files map[string]string) string {
	for name := range files {
		lower := strings.ToLower(name)
		switch {
		case lower == "go.mod":
			return "go"
		case lower == "package.json":
			return "nodejs"
		case lower == "requirements.txt" || lower == "main.py" || lower == "app.py":
			return "python"
		}
	}
	// Check for HTML files (static site)
	for name := range files {
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".html") {
			return "static"
		}
	}
	return "static"
}

// GenerateDockerfile returns a Dockerfile for the given language.
// If language is empty or "auto", it auto-detects from the file map.
func GenerateDockerfile(language string, files map[string]string) string {
	if language == "" || language == "auto" {
		language = DetectLanguage(files)
	}

	switch language {
	case "nodejs":
		return dockerfileNodeJS()
	case "python":
		return dockerfilePython()
	case "go":
		return dockerfileGo()
	default:
		return dockerfileStatic()
	}
}

func dockerfileStatic() string {
	return `FROM nginx:alpine
RUN sed -i 's/listen\s*80;/listen 8080;/g' /etc/nginx/conf.d/default.conf
COPY . /usr/share/nginx/html
EXPOSE 8080
`
}

func dockerfileNodeJS() string {
	return `FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci --production 2>/dev/null || npm install --production
COPY . .
RUN npm run build 2>/dev/null || true

FROM node:22-alpine
WORKDIR /app
COPY --from=build /app .
EXPOSE 8080
CMD ["node", "index.js"]
`
}

func dockerfilePython() string {
	return `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt* ./
RUN pip install --no-cache-dir -r requirements.txt 2>/dev/null || true
COPY . .
EXPOSE 8080
CMD ["python", "main.py"]
`
}

func dockerfileGo() string {
	return `FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -o server .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/server .
EXPOSE 8080
CMD ["./server"]
`
}
