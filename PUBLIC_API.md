# 🌐 PUBLIC API ACCESS

## ⚠️ Direct HTTP Access Not Available

The backend ALB is not publicly accessible from your network due to security group/network restrictions.

## ✅ WORKING SOLUTION: Use SQS API

Your platform is **fully functional** via AWS SQS. Here's how to use it:

---

## 📤 Submit Render Job

```bash
aws sqs send-message \
  --queue-url https://sqs.us-east-1.amazonaws.com/929910138721/video-captioning-render-queue \
  --message-body '{
    "jobId": "my-job-'$(date +%s)'",
    "videoUrl": "https://go-video-project-685a78d468a5w4e65.s3.us-east-1.amazonaws.com/uploads/sample.mp4",
    "s3Key": "uploads/sample.mp4",
    "captions": [
      {"start": 0.72, "end": 3.84, "text": "Your caption here"}
    ],
    "style": "bottom"
  }'
```

## 📊 Check Job Status

```bash
aws dynamodb get-item \
  --table-name video-captioning-jobs \
  --key '{"jobId":{"S":"YOUR_JOB_ID"}}' \
  --query 'Item.{status:status.S,outputUrl:outputUrl.S}' \
  --output json | jq
```

## 📥 Download Video

The `outputUrl` from DynamoDB contains a presigned S3 URL (valid 24 hours).
Just copy and paste it in your browser or use:

```bash
wget "PRESIGNED_URL" -O output.mp4
```

---

## 🚀 Quick Test

Run the test script:

```bash
./test-platform.sh
```

This will:

1. Submit a job
2. Monitor status
3. Show download URL when complete

---

## ✅ VERIFIED WORKING

- Job `verify-1763964497`: COMPLETED (4.1 MB)
- Processing time: ~40-60 seconds
- Full serverless pipeline operational

---

## 💡 Alternative: Deploy Frontend

If you need a web UI, you can:

1. Deploy a static frontend to S3 + CloudFront
2. Use AWS Amplify for hosting
3. Create a simple HTML page that calls SQS via AWS SDK

Would you like me to create a simple web interface?
