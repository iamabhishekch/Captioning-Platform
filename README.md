# 🎬 Remotion Captioning Platform

> A minimal, fully-functional video captioning platform with Hinglish support

Built with **Go** (backend) + **AssemblyAI** (speech-to-text) + **Remotion** (video rendering)

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Node](https://img.shields.io/badge/Node-18+-339933?style=flat&logo=node.js)](https://nodejs.org/)
[![Remotion](https://img.shields.io/badge/Remotion-4.0-FF6154?style=flat)](https://remotion.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## ✨ Features

- 📤 Upload MP4 videos
- 🎤 Auto-generate captions using AssemblyAI
- 🌏 Hinglish support (Hindi Devanagari + English)
- ✏️ Edit captions before rendering
- 🎨 3 caption styles: Bottom, Top Bar, Karaoke
- 🎬 Preview and export captioned videos
- 🚀 Deployable to Render.com
- 💻 CLI render fallback

## 🚀 Quick Start

**👉 New to the project? See [START-HERE.md](START-HERE.md) or [QUICKSTART.md](QUICKSTART.md) for a 5-minute setup guide!**

### Automated Setup (Recommended)

**Linux/Mac:**
```bash
chmod +x setup.sh
./setup.sh
```

**Windows:**
```cmd
setup.bat
```

Then edit `.env` and add your AssemblyAI API key.

### Manual Setup

1. **Clone and configure:**
```bash
git clone <your-repo>
cd remotion-captioning-platform
cp .env.example .env
# Edit .env and add ASSEMBLYAI_KEY
```

2. **Install Go dependencies:**
```bash
cd backend-go
go mod download
cd ..
```

3. **Install Node dependencies:**
```bash
cd remotion-app
npm install
cd ..
```

4. **Download fonts (for Hinglish):**
```bash
cd remotion-app/public/fonts
wget https://github.com/notofonts/noto-fonts/raw/main/hinted/ttf/NotoSans/NotoSans-Regular.ttf
wget https://github.com/notofonts/noto-fonts/raw/main/hinted/ttf/NotoSansDevanagari/NotoSansDevanagari-Regular.ttf
cd ../../..
```

## 📋 Prerequisites

- **Go**: 1.21 or higher
- **Node.js**: 18 or higher
- **AssemblyAI API Key**: Get free key at https://www.assemblyai.com/

## 🔧 Environment Variables

Create a `.env` file in the root:

```env
# Required
ASSEMBLYAI_KEY=your_assemblyai_api_key_here

# Optional
RENDER_REMOTION_URL=http://localhost:3000

# Optional: S3 Storage
S3_BUCKET=your-bucket-name
AWS_ACCESS_KEY_ID=your-aws-key
AWS_SECRET_ACCESS_KEY=your-aws-secret
```

## 🏃 Running Locally

You need two terminals running simultaneously:

**Terminal 1 - Go Backend:**
```bash
cd backend-go
go run main.go
```
Backend runs on `http://localhost:7070`

**Terminal 2 - Remotion Service:**
```bash
cd remotion-app
npm run server
```
Remotion service runs on `http://localhost:3000`

**Open Browser:**
Navigate to `http://localhost:7070` and start captioning!

## 🧪 Testing

Run health checks:
```bash
# Linux/Mac
./test-api.sh

# Windows
test-api.bat
```

Or test manually:
1. Upload a video file (.mp4)
2. Click "Auto-generate Captions"
3. Edit captions if needed
4. Select a caption style
5. Click "Render Final MP4"
6. Download your captioned video

## 💻 CLI Render Fallback

If the Remotion server is unavailable, you can render directly via CLI:

```bash
cd remotion-app
npx remotion render src/index.tsx CaptionedVideo out/final.mp4 \
  --props '{"videoUrl":"../uploads/video.mp4","captions":[{"start":0,"end":2,"text":"नमस्ते Hello"}],"style":"bottom"}'
```

The web UI will provide the exact command if remote rendering fails.

## 🚀 Deployment to Render

See detailed guide in [DEPLOYMENT.md](DEPLOYMENT.md)

**Quick Summary:**

1. **Deploy Remotion Service:**
   - Root Directory: `remotion-app`
   - Build: `npm install`
   - Start: `node server/index.js`

2. **Deploy Go Backend:**
   - Root Directory: `backend-go`
   - Build: `go build -o main .`
   - Start: `./main`
   - Add env vars: `ASSEMBLYAI_KEY`, `RENDER_REMOTION_URL`

**Cost:** Free tier available, ~$14/mo for production

## 📁 Project Structure

```
/backend-go       - Go server (port 7070)
  ├── main.go     - Main server with all endpoints
  └── templates/  - HTML UI (htmx + Tailwind)

/remotion-app     - Node/Remotion rendering service (port 3000)
  ├── server/     - Express render service
  ├── src/        - Remotion compositions
  └── public/     - Fonts for Hinglish

/samples          - Sample videos
```

See [PROJECT-STRUCTURE.md](PROJECT-STRUCTURE.md) for complete details.

## 📚 Documentation

**📑 [INDEX.md](INDEX.md) - Quick navigation to all files**  
**📖 [DOCUMENTATION-INDEX.md](DOCUMENTATION-INDEX.md) - Complete documentation guide**

Quick links:
- [QUICKSTART.md](QUICKSTART.md) - 5-minute setup guide ⭐
- [API-EXAMPLES.md](API-EXAMPLES.md) - Complete API reference
- [DEPLOYMENT.md](DEPLOYMENT.md) - Render.com deployment guide
- [PROJECT-STRUCTURE.md](PROJECT-STRUCTURE.md) - Codebase overview
- [REQUIREMENTS-CHECKLIST.md](REQUIREMENTS-CHECKLIST.md) - Feature verification
- [README-backend.md](backend-go/README-backend.md) - Go backend details
- [README-remotion.md](remotion-app/README-remotion.md) - Remotion service details

## 🎨 Caption Styles

1. **Bottom** - Centered white text with black outline (classic subtitles)
2. **Top Bar** - Black bar at top with white text (news-style)
3. **Karaoke** - Progressive word highlighting in gold (sing-along style)

## 🌏 Hinglish Support

The platform supports mixed Hindi (Devanagari) and English text:
- "नमस्ते, welcome to our platform"
- "यह video बहुत अच्छा है"
- "Let's start the tutorial अभी"

Fonts used: Noto Sans + Noto Sans Devanagari

## 📝 Sample Files

Place your test video at `samples/sample.mp4` (with Hinglish audio).

Generate captioned output:
```bash
cd remotion-app
npx remotion render src/index.tsx CaptionedVideo ../samples/sample-captioned.mp4 \
  --props '{"videoUrl":"../samples/sample.mp4","captions":[{"start":0,"end":2,"text":"नमस्ते Hello"}],"style":"bottom"}'
```

## 🛠️ Tech Stack

- **Backend**: Go 1.21, Gin framework
- **Frontend**: HTML, htmx, Tailwind CSS
- **Transcription**: AssemblyAI API
- **Rendering**: Remotion (React-based video)
- **Fonts**: Google Noto Sans family

## 📄 License

MIT License - feel free to use for your projects!

## 🤝 Contributing

This is a minimal submission for an assignment. For production use, consider:
- Rate limiting
- User authentication
- Job queue (Redis)
- Cloud storage (S3)
- Real-time preview
- Batch processing

## 🐛 Troubleshooting

**Fonts not loading:**
- Download fonts manually from `remotion-app/public/fonts/README.md`

**AssemblyAI fails:**
- Check your API key in `.env`
- Verify video has audio track

**Render times out:**
- Use CLI fallback command provided in UI
- Upgrade Render instance for faster rendering

**Port already in use:**
- Change port in code or kill existing process
