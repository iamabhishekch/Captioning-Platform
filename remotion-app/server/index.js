// Minimal Express server for Remotion rendering
const express = require("express");
const cors = require("cors");
const { spawn } = require("child_process");
const path = require("path");
const fs = require("fs");

const app = express();
const PORT = process.env.PORT || 3000;

app.use(cors());
app.use(express.json({ limit: "50mb" }));

// Ensure output directory exists
const outDir = path.join(__dirname, "..", "out");
if (!fs.existsSync(outDir)) {
  fs.mkdirSync(outDir, { recursive: true });
}

/**
 * POST /render
 * Accepts: { videoUrl, captions, style, outPath }
 * Runs: npx remotion render src/index.tsx CaptionedVideo <outPath> --props-file <file>
 * Returns: { success, outPath, logs }
 */
app.post("/render", async (req, res) => {
  // FIX #5: Simple authentication
  const apiKey = req.headers["x-api-key"];
  const expectedKey = process.env.RENDER_API_KEY;

  if (expectedKey && apiKey !== expectedKey) {
    return res.status(401).json({
      success: false,
      error: "Unauthorized: Invalid or missing API key",
    });
  }

  const { videoUrl, captions, style, outPath } = req.body;

  if (!videoUrl || !captions || !style) {
    return res.status(400).json({
      success: false,
      error: "Missing required fields: videoUrl, captions, style",
    });
  }

  // FIX #4: Validate inputs before spawning
  const validStyles = ["bottom", "top-bar", "karaoke"];
  if (!validStyles.includes(style)) {
    return res.status(400).json({
      success: false,
      error: "Invalid style. Must be: bottom, top-bar, or karaoke",
    });
  }

  if (!Array.isArray(captions)) {
    return res.status(400).json({
      success: false,
      error: "Captions must be an array",
    });
  }

  // Validate videoUrl doesn't contain path traversal
  if (videoUrl.includes("..") || videoUrl.includes("~")) {
    return res.status(400).json({
      success: false,
      error: "Invalid videoUrl",
    });
  }

  // Default output path
  const outputPath = outPath || `out/video_${Date.now()}.mp4`;
  const fullOutputPath = path.join(__dirname, "..", outputPath);

  // FIX #4: Write props to temporary file instead of passing in CLI
  const propsFileName = `props-${Date.now()}.json`;
  const propsFile = path.join(__dirname, "..", propsFileName);

  // Convert relative path to absolute path for video
  // Video is in backend-go/uploads/, which is at project root level
  let absoluteVideoPath;

  console.log("=== VIDEO PATH DEBUGGING ===");
  console.log("1. Raw videoUrl received:", videoUrl);
  console.log("2. __dirname:", __dirname);
  console.log("3. Is absolute?", path.isAbsolute(videoUrl));

  if (path.isAbsolute(videoUrl)) {
    absoluteVideoPath = videoUrl;
    console.log("4. Using absolute path as-is");
  } else {
    // videoUrl is like "uploads/file.mp4"
    // Go up from remotion-app/server to project root, then into backend-go
    absoluteVideoPath = path.resolve(
      __dirname,
      "..",
      "..",
      "backend-go",
      videoUrl
    );
    console.log("4. Resolved relative path");
  }

  console.log("5. Final absolute path:", absoluteVideoPath);
  console.log("6. File exists?", fs.existsSync(absoluteVideoPath));

  if (!fs.existsSync(absoluteVideoPath)) {
    console.error("ERROR: Video file not found!");
    // Try to find where it might be
    const altPath1 = path.resolve(__dirname, "..", videoUrl);
    const altPath2 = path.resolve(__dirname, "..", "..", videoUrl);
    console.log("7. Checking alternative paths:");
    console.log("   - remotion-app/" + videoUrl + ":", fs.existsSync(altPath1));
    console.log("   - project-root/" + videoUrl + ":", fs.existsSync(altPath2));
  }
  console.log("=== END DEBUGGING ===");

  // Check if videoUrl is already a full HTTP/HTTPS URL (S3, etc)
  let videoUrlForRemotion;
  if (videoUrl.startsWith("http://") || videoUrl.startsWith("https://")) {
    // Already a full URL (S3, presigned URL, etc) - use as is
    videoUrlForRemotion = videoUrl;
    console.log("Video URL for Remotion (Direct S3):", videoUrlForRemotion);
  } else {
    // Relative path - prepend backend URL
    const backendURL = process.env.BACKEND_URL || "http://localhost:7070";
    videoUrlForRemotion = `${backendURL}/${videoUrl.replace(/\\/g, "/")}`;
    console.log("Video URL for Remotion (via Backend):", videoUrlForRemotion);
  }

  const props = { videoUrl: videoUrlForRemotion, captions, style };

  try {
    fs.writeFileSync(propsFile, JSON.stringify(props));
  } catch (err) {
    return res.status(500).json({
      success: false,
      error: "Failed to write props file",
    });
  }

  // FIX #4: Build command - use absolute path for props file
  const args = [
    "remotion",
    "render",
    "src/index.tsx",
    "CaptionedVideo",
    fullOutputPath,
    "--props",
    propsFile, // Use absolute path
  ];

  console.log("Starting render with command:", "npx", args.join(" "));

  // FIX #4: Spawn child process - use shell:true on Windows for npx to work
  const renderProcess = spawn("npx", args, {
    cwd: path.join(__dirname, ".."),
    shell: true, // Required on Windows for npx to be found
  });

  let logs = "";

  // Capture stdout
  renderProcess.stdout.on("data", (data) => {
    const message = data.toString();
    logs += message;
    console.log(message);
  });

  // Capture stderr
  renderProcess.stderr.on("data", (data) => {
    const message = data.toString();
    logs += message;
    console.error(message);
  });

  // Handle completion
  renderProcess.on("close", (code) => {
    // FIX #4: Delete temporary props file
    // TEMPORARILY DISABLED FOR DEBUGGING
    // try {
    //   fs.unlinkSync(propsFile);
    // } catch (err) {
    //   console.error('Failed to delete props file:', err);
    // }
    console.log("Props file kept for debugging:", propsFile);

    if (code === 0) {
      console.log("Render completed successfully");
      res.json({
        success: true,
        outPath: outputPath,
        outUrl: `/download/${path.basename(outputPath)}`,
        logs,
      });
    } else {
      console.error("Render failed with code:", code);
      res.status(500).json({
        success: false,
        error: `Render process exited with code ${code}`,
        logs,
      });
    }
  });

  // Handle errors
  renderProcess.on("error", (error) => {
    // FIX #4: Delete temporary props file on error
    // TEMPORARILY DISABLED FOR DEBUGGING
    // try {
    //   fs.unlinkSync(propsFile);
    // } catch (err) {
    //   console.error('Failed to delete props file:', err);
    // }
    console.log("Props file kept for debugging:", propsFile);

    console.error("Render process error:", error);
    res.status(500).json({
      success: false,
      error: error.message,
      logs,
    });
  });
});

/**
 * GET /download/:filename
 * Serves rendered video files
 */
app.get("/download/:filename", (req, res) => {
  const filename = req.params.filename;
  const filePath = path.join(outDir, filename);

  if (fs.existsSync(filePath)) {
    res.download(filePath);
  } else {
    res.status(404).json({ error: "File not found" });
  }
});

/**
 * GET /health
 * Health check endpoint
 */
app.get("/health", (req, res) => {
  res.json({ status: "ok", service: "remotion-render" });
});

app.listen(PORT, () => {
  console.log(`Remotion render service running on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
});
