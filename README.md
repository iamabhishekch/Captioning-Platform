# Video Captioning Platform

A production-ready video captioning platform with Hinglish support, built with Go, AssemblyAI, and Remotion.

## Author

Abhishek Chaurasiya

## Overview

This platform automatically generates and renders captions for videos with support for mixed Hindi-English (Hinglish) content. Upload a video, generate captions using AI, edit them if needed, and export with professional styling.

## Features

- Upload MP4 videos to AWS S3
- Auto-generate captions using AssemblyAI
- Support for Hinglish (Hindi Devanagari + English)
- Edit captions before rendering
- Three caption styles: Bottom, Top Bar, Karaoke
- Preview videos with presigned URLs
- Export captioned videos

## Tech Stack

- Backend: Go 1.21 with Gin framework
- Frontend: HTML, htmx, Tailwind CSS
- Transcription: AssemblyAI API
- Video Rendering: Remotion (React-based)
- Storage: AWS S3
- Deployment: AWS ECS Fargate
- CI/CD: GitHub Actions

## Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- AssemblyAI API key
- AWS account with S3 bucket

## Environment Variables

Create a `.env` file:

```env
ASSEMBLYAI_KEY=your_assemblyai_api_key
S3_BUCKET=your-bucket-name
AWS_ACCESS_KEY_ID=your-aws-key
AWS_SECRET_ACCESS_KEY=your-aws-secret
AWS_REGION=us-east-1
RENDER_REMOTION_URL=http://localhost:3000
RENDER_API_KEY=secure_key_12345
```

## Local Development

### Using Docker Compose (Recommended)

```bash
docker-compose up -d
```

Access at http://localhost:7070

### Manual Setup

Terminal 1 - Backend:

```bash
cd backend-go
go run main.go
```

Terminal 2 - Remotion Service:

```bash
cd remotion-app
npm run server
```

## Deployment

### AWS ECS Fargate

The application is deployed on AWS ECS Fargate with automatic updates via GitHub Actions.

Live URL: http://44.202.110.90:7070

Services:

- Backend: 1 vCPU, 2GB RAM
- Remotion: 2 vCPU, 4GB RAM

### CI/CD Pipeline

Every push to main branch:

1. Builds Docker images
2. Pushes to Docker Hub
3. Deploys to AWS ECS Fargate

GitHub Secrets required:

- DOCKER_USERNAME
- DOCKER_PASSWORD
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY

## Project Structure

```
/backend-go           - Go backend server
  ├── main.go         - API endpoints
  ├── templates/      - HTML UI
  └── Dockerfile      - Backend container

/remotion-app         - Remotion rendering service
  ├── server/         - Express server
  ├── src/            - Remotion compositions
  └── Dockerfile      - Remotion container

/.github/workflows    - CI/CD pipelines
```

## API Endpoints

- POST /upload - Upload video to S3
- POST /get-presigned-url - Get presigned URL for preview
- POST /transcribe - Generate captions with AssemblyAI
- POST /render-job - Render video with captions
- GET /health - Health check

## Caption Styles

1. Bottom - Classic centered subtitles with outline
2. Top Bar - News-style top bar with text
3. Karaoke - Word-by-word highlighting

## Contact

Abhishek Chaurasiya
Email: chaurasiyaa750@gmail.com
