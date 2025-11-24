const AWS = require("aws-sdk");
const https = require("https");
const http = require("http");

const dynamodb = new AWS.DynamoDB.DocumentClient();
const s3 = new AWS.S3();

const DYNAMODB_TABLE = process.env.DYNAMODB_TABLE;
const S3_BUCKET = process.env.S3_BUCKET;
const REMOTION_URL = process.env.REMOTION_URL || "http://remotion-service:3000";
const RENDER_API_KEY = process.env.RENDER_API_KEY;

exports.handler = async (event) => {
  console.log("Received event:", JSON.stringify(event, null, 2));

  for (const record of event.Records) {
    const message = JSON.parse(record.body);
    const { jobId, videoUrl, captions, style, s3Key } = message;

    console.log(`Processing job ${jobId}`);

    try {
      // Update job status to processing
      await updateJobStatus(jobId, "processing");

      // Generate presigned URL for input video (valid for 1 hour)
      const presignedInputUrl = await getPresignedUrl(s3Key, 3600);
      console.log(
        `Generated presigned URL for input video: ${presignedInputUrl.substring(
          0,
          100
        )}...`
      );

      // Trigger Remotion render with presigned URL
      const renderResult = await triggerRender(
        presignedInputUrl,
        captions,
        style,
        jobId
      );

      if (!renderResult.success) {
        throw new Error(renderResult.error || "Render failed");
      }

      // Download rendered video from Remotion
      const videoBuffer = await downloadVideo(renderResult.downloadUrl);

      // Upload to S3
      const outputKey = `output/video_${jobId}.mp4`;
      await uploadToS3(videoBuffer, outputKey);

      // Generate presigned URL
      const presignedUrl = await getPresignedUrl(outputKey);

      // Update job status to completed
      await updateJobStatus(jobId, "completed", presignedUrl);

      console.log(`Job ${jobId} completed successfully`);
    } catch (error) {
      console.error(`Job ${jobId} failed:`, error);
      await updateJobStatus(jobId, "failed", null, error.message);
    }
  }

  return { statusCode: 200, body: "Processing complete" };
};

async function updateJobStatus(jobId, status, outputUrl = null, error = null) {
  const params = {
    TableName: DYNAMODB_TABLE,
    Key: { jobId },
    UpdateExpression:
      "SET #status = :status, updatedAt = :updatedAt" +
      (outputUrl ? ", outputUrl = :outputUrl" : "") +
      (error ? ", #error = :error" : ""),
    ExpressionAttributeNames: {
      "#status": "status",
      ...(error && { "#error": "error" }),
    },
    ExpressionAttributeValues: {
      ":status": status,
      ":updatedAt": new Date().toISOString(),
      ...(outputUrl && { ":outputUrl": outputUrl }),
      ...(error && { ":error": error }),
    },
  };

  await dynamodb.update(params).promise();
}

async function triggerRender(videoUrl, captions, style, jobId) {
  const renderRequest = {
    videoUrl,
    captions,
    style,
    outPath: `out/video_${jobId}.mp4`,
  };

  return new Promise((resolve, reject) => {
    const data = JSON.stringify(renderRequest);
    const url = new URL(REMOTION_URL + "/render");

    const options = {
      hostname: url.hostname,
      port: url.port || (url.protocol === "https:" ? 443 : 80),
      path: url.pathname,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Content-Length": data.length,
        "x-api-key": RENDER_API_KEY,
      },
      timeout: 600000, // 10 minutes
    };

    const protocol = url.protocol === "https:" ? https : http;
    const req = protocol.request(options, (res) => {
      let body = "";
      res.on("data", (chunk) => (body += chunk));
      res.on("end", () => {
        try {
          const result = JSON.parse(body);
          if (result.success) {
            result.downloadUrl = `${REMOTION_URL}/download/${result.outPath
              .split("/")
              .pop()}`;
          }
          resolve(result);
        } catch (e) {
          reject(new Error("Invalid response from Remotion service"));
        }
      });
    });

    req.on("error", reject);
    req.on("timeout", () => {
      req.destroy();
      reject(new Error("Render request timed out"));
    });
    req.write(data);
    req.end();
  });
}

async function downloadVideo(url) {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);
    const protocol = urlObj.protocol === "https:" ? https : http;

    protocol
      .get(url, (res) => {
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function uploadToS3(buffer, key) {
  const params = {
    Bucket: S3_BUCKET,
    Key: key,
    Body: buffer,
    ContentType: "video/mp4",
  };

  await s3.putObject(params).promise();
}

async function getPresignedUrl(key, expiresIn = 86400) {
  const params = {
    Bucket: S3_BUCKET,
    Key: key,
    Expires: expiresIn, // Default 24 hours
  };

  return s3.getSignedUrl("getObject", params);
}
