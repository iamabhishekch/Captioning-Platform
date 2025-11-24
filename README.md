# Video Captioning Platform

A serverless video captioning platform that automatically generates and renders captions for videos using AI.

**Author:** Abhishek Chaurasiya

## Live Demo

🌐 **Frontend:** http://video-captioning-frontend-a02911db.s3-website-us-east-1.amazonaws.com

## What It Does

Upload a video → AI generates captions → Edit if needed → Render with captions → Download

## Architecture

**Serverless AWS Stack:**

- **Frontend:** S3 Static Website (HTML/CSS/JS)
- **Backend API:** Go on ECS Fargate (auto-scales)
- **Video Rendering:** Remotion on ECS Fargate
- **Async Processing:** Lambda + SQS Queue
- **Storage:** S3 for videos
- **Database:** DynamoDB for job tracking
- **Load Balancer:** ALB for traffic distribution

**Why This Stack:**

- **Go Backend:** Fast, efficient, single binary deployment
- **ECS Fargate:** Serverless containers, no server management
- **Lambda + SQS:** Async processing, scales automatically
- **S3 + DynamoDB:** Cheap storage and fast queries
- **Remotion:** React-based video rendering with precise control

## Features

- Upload MP4 videos
- Auto-generate captions using AssemblyAI
- Support for Hinglish (Hindi + English)
- Edit captions before rendering
- Three caption styles: Bottom, Top Bar, Karaoke
- Real-time job status tracking
- Preview and download rendered videos

## Tech Stack

**Backend:**

- Go 1.21 with Gin framework
- AWS SDK for S3, SQS, DynamoDB
- AssemblyAI for transcription

**Frontend:**

- HTML, Tailwind CSS, Vanilla JavaScript
- Hosted on S3 as static website

**Video Rendering:**

- Remotion (React + TypeScript)
- Node.js 18 Express server

**Infrastructure:**

- Terraform for IaC
- GitHub Actions for CI/CD
- AWS: ECS, Lambda, S3, SQS, DynamoDB, ALB

## Deployment

**Fully Automated CI/CD:**

Push to `main` or `dev/aws-deploy` branch triggers:

1. Backend Docker build → Push to ECR → Deploy to ECS
2. Remotion Docker build → Push to ECR → Deploy to ECS
3. Lambda function package → Deploy to AWS Lambda
4. Frontend templates → Sync to S3

**Infrastructure:**

- Managed by Terraform
- One-time setup, then automated deployments

## Local Development

### Using Docker Compose

```bash
# Create .env file with your credentials
cp .env.example .env

# Start all services
docker-compose up -d

# Access at http://localhost:7070
```

### Manual Setup

**Terminal 1 - Backend:**

```bash
cd backend-go
go run main.go
```

**Terminal 2 - Remotion:**

```bash
cd remotion-app
npm run server
```

## Environment Variables

```env
# Required
ASSEMBLYAI_KEY=your_assemblyai_api_key
S3_BUCKET=your-bucket-name
AWS_REGION=us-east-1

# AWS Mode (Production)
SQS_QUEUE_URL=https://sqs.us-east-1.amazonaws.com/ACCOUNT/queue-name
DYNAMODB_TABLE=video-captioning-jobs

# Local Mode (Docker)
RENDER_REMOTION_URL=http://remotion-service:3000
RENDER_API_KEY=secure_key_12345
```

## Project Structure

```
/backend-go              - Go backend API
  ├── main.go            - API endpoints & logic
  ├── templates/         - Frontend HTML
  └── Dockerfile         - Container image

/remotion-app            - Video rendering service
  ├── server/            - Express API server
  ├── src/               - Remotion compositions
  └── Dockerfile         - Container image

/lambda-worker           - Async job processor
  └── index.js           - Lambda handler

/terraform               - Infrastructure as Code
  └── main.tf            - AWS resources

/.github/workflows       - CI/CD pipelines
```

## API Endpoints

- `POST /upload` - Upload video to S3
- `POST /transcribe` - Generate captions with AI
- `POST /render-job` - Create render job
- `GET /render-job/:id` - Check job status
- `POST /get-presigned-url` - Get video preview URL
- `GET /health` - Health check

## Caption Styles

1. **Bottom** - Classic centered subtitles
2. **Top Bar** - News-style top banner
3. **Karaoke** - Word-by-word highlighting

## Contact

**Abhishek Chaurasiya**  
Email: chaurasiyaa750@gmail.com
