#!/bin/bash
set -e

echo "🎬 Testing Serverless Video Captioning Platform"
echo "================================================"
echo ""

JOB_ID="test-$(date +%s)"
echo "📝 Job ID: $JOB_ID"
echo ""

# Submit job
echo "📤 Submitting render job to SQS..."
aws sqs send-message \
  --queue-url https://sqs.us-east-1.amazonaws.com/929910138721/video-captioning-render-queue \
  --message-body "{
    \"jobId\":\"$JOB_ID\",
    \"videoUrl\":\"https://go-video-project-685a78d468a5w4e65.s3.us-east-1.amazonaws.com/uploads/sample.mp4\",
    \"s3Key\":\"uploads/sample.mp4\",
    \"captions\":[
      {\"start\":0.72,\"end\":3.84,\"text\":\"Big man in a suit of armor. Take\"},
      {\"start\":3.84,\"end\":6.72,\"text\":\"that off. What are you? Genius, billionaire, playboy,\"},
      {\"start\":6.72,\"end\":9.44,\"text\":\"philanthropist. I know guys with none of that\"}
    ],
    \"style\":\"bottom\"
  }" > /dev/null

echo "✅ Job submitted successfully"
echo ""
echo "⏳ Monitoring job status (this takes ~40-60 seconds)..."
echo ""

# Monitor status
for i in {1..40}; do
  STATUS=$(aws dynamodb get-item \
    --table-name video-captioning-jobs \
    --key "{\"jobId\":{\"S\":\"$JOB_ID\"}}" \
    --query 'Item.status.S' \
    --output text 2>/dev/null)
  
  if [ -z "$STATUS" ] || [ "$STATUS" = "None" ]; then
    echo "[$i] Waiting for job to start..."
  else
    echo "[$i] Status: $STATUS"
  fi
  
  if [ "$STATUS" = "completed" ]; then
    echo ""
    echo "🎉 SUCCESS! Video rendered successfully"
    echo ""
    
    OUTPUT_URL=$(aws dynamodb get-item \
      --table-name video-captioning-jobs \
      --key "{\"jobId\":{\"S\":\"$JOB_ID\"}}" \
      --query 'Item.outputUrl.S' \
      --output text)
    
    echo "📥 Download URL (valid for 24 hours):"
    echo "$OUTPUT_URL"
    echo ""
    
    # Check file size
    FILE_SIZE=$(aws s3 ls s3://go-video-project-685a78d468a5w4e65/output/video_$JOB_ID.mp4 --human-readable | awk '{print $3, $4}')
    echo "📦 File size: $FILE_SIZE"
    echo ""
    
    exit 0
  elif [ "$STATUS" = "failed" ]; then
    echo ""
    echo "❌ FAILED"
    ERROR=$(aws dynamodb get-item \
      --table-name video-captioning-jobs \
      --key "{\"jobId\":{\"S\":\"$JOB_ID\"}}" \
      --query 'Item.error.S' \
      --output text)
    echo "Error: $ERROR"
    echo ""
    exit 1
  fi
  
  sleep 10
done

echo ""
echo "⏱️ Timeout: Job is still processing. Check status manually:"
echo "aws dynamodb get-item --table-name video-captioning-jobs --key '{\"jobId\":{\"S\":\"$JOB_ID\"}}' --query 'Item.status.S' --output text"
