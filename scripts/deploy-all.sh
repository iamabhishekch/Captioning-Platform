#!/bin/bash
set -e

echo "🚀 Deploying all services..."

AWS_REGION=${AWS_REGION:-us-east-1}
ECR_REGISTRY=$(aws ecr describe-repositories --repository-names video-captioning-backend --region $AWS_REGION --query 'repositories[0].repositoryUri' --output text | cut -d'/' -f1)

# Deploy Backend
echo "📦 Building and deploying Backend..."
docker buildx build --platform linux/amd64 \
  -t $ECR_REGISTRY/video-captioning-backend:latest \
  -f backend-go/Dockerfile backend-go --push

aws ecs update-service \
  --cluster video-captioning-cluster \
  --service video-captioning-backend \
  --force-new-deployment \
  --region $AWS_REGION > /dev/null

echo "✅ Backend deployed"

# Deploy Remotion
echo "📦 Building and deploying Remotion..."
docker buildx build --platform linux/amd64 \
  -t $ECR_REGISTRY/video-captioning-remotion:latest \
  -f remotion-app/Dockerfile remotion-app --push

aws ecs update-service \
  --cluster video-captioning-cluster \
  --service video-captioning-remotion \
  --force-new-deployment \
  --region $AWS_REGION > /dev/null

echo "✅ Remotion deployed"

# Deploy Lambda
echo "📦 Deploying Lambda..."
cd lambda-worker
zip -r ../lambda-worker.zip . -q
cd ..

aws lambda update-function-code \
  --function-name video-captioning-render-worker \
  --zip-file fileb://lambda-worker.zip \
  --region $AWS_REGION > /dev/null

rm lambda-worker.zip

echo "✅ Lambda deployed"
echo ""
echo "🎉 All services deployed successfully!"
