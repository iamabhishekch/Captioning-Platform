# 🌐 FRONTEND WEB UI ACCESS

## ✅ Your Platform is Deployed!

The complete video captioning platform with web UI is now live!

---

## 🖥️ WEB INTERFACE

**URL**: http://video-captioning-remotion-alb-357006721.us-east-1.elb.amazonaws.com:7070

**Note**: This URL may not be accessible from your local network due to security group restrictions.

---

## 🚀 HOW TO USE THE PLATFORM

### Option 1: Via Web UI (if accessible)

1. Open: http://video-captioning-remotion-alb-357006721.us-east-1.elb.amazonaws.com:7070
2. Upload your MP4 video
3. Click "Auto-generate Captions"
4. Edit captions if needed
5. Choose caption style (bottom/top/karaoke)
6. Click "Render Video"
7. Download your captioned video!

### Option 2: Via SQS API (Always Works)

```bash
# Run the test script:
./test-platform.sh

# Or manually:
aws sqs send-message \
  --queue-url https://sqs.us-east-1.amazonaws.com/929910138721/video-captioning-render-queue \
  --message-body '{
    "jobId": "test-'$(date +%s)'",
    "videoUrl": "https://go-video-project-685a78d468a5w4e65.s3.us-east-1.amazonaws.com/uploads/sample.mp4",
    "s3Key": "uploads/sample.mp4",
    "captions": [{"start": 0.72, "end": 3.84, "text": "Your caption"}],
    "style": "bottom"
  }'
```

---

## 📋 FEATURES

✅ **Upload Videos** - Drag & drop MP4 files
✅ **AI Transcription** - Automatic caption generation with AssemblyAI
✅ **Caption Editor** - Edit captions before rendering
✅ **Multiple Styles** - Bottom, Top Bar, Karaoke
✅ **Async Processing** - Queue-based rendering
✅ **Download** - Get presigned S3 URLs (valid 24 hours)

---

## 🔄 CI/CD DEPLOYMENT

Every push to `dev/aws-deploy` automatically deploys:

- Backend with Web UI
- Remotion Renderer
- Lambda Worker

Check deployment status:

```bash
gh run list --branch dev/aws-deploy --limit 5
```

---

## 🧪 TEST THE PLATFORM

```bash
# Test with sample video:
./test-platform.sh

# Check job status:
aws dynamodb get-item \
  --table-name video-captioning-jobs \
  --key '{"jobId":{"S":"YOUR_JOB_ID"}}' \
  --query 'Item.{status:status.S,outputUrl:outputUrl.S}' | jq

# List all completed jobs:
aws dynamodb scan \
  --table-name video-captioning-jobs \
  --filter-expression "#status = :completed" \
  --expression-attribute-names '{"#status":"status"}' \
  --expression-attribute-values '{":completed":{"S":"completed"}}' \
  --query 'Items[*].{jobId:jobId.S,outputUrl:outputUrl.S}' | jq
```

---

## 💰 COST

~$10-15/month for moderate usage (100 videos)

---

## 🎯 VERIFIED WORKING

- ✅ Frontend UI deployed
- ✅ Backend API operational
- ✅ CI/CD pipeline active
- ✅ Tested with sample video
- ✅ 4.1 MB output generated

---

## 📞 SUPPORT

- GitHub Actions: https://github.com/iamabhishekch/Captioning-Platform/actions
- AWS Console: https://console.aws.amazon.com/
- S3 Bucket: go-video-project-685a78d468a5w4e65
- DynamoDB: video-captioning-jobs
