package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// Caption represents a single caption with timing
type Caption struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// AssemblyAI response structures
type AssemblyAIUploadResponse struct {
	UploadURL string `json:"upload_url"`
}

type AssemblyAITranscriptRequest struct {
	AudioURL string `json:"audio_url"`
}

type AssemblyAITranscriptResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Words  []struct {
		Text  string  `json:"text"`
		Start int     `json:"start"`
		End   int     `json:"end"`
	} `json:"words"`
}

// uploadToS3FromReader uploads data from an io.Reader to S3 bucket
func uploadToS3FromReader(reader io.Reader, bucketName, key, contentType string) (string, error) {
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsRegion := os.Getenv("AWS_REGION")
	
	if awsRegion == "" {
		awsRegion = "us-east-1"
	}
	
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create AWS session: %v", err)
	}
	
	// Create S3 service client
	svc := s3.New(sess)
	
	// Read all data into memory for upload
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read data: %v", err)
	}
	
	// Upload to S3
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}
	
	// Generate public URL
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, awsRegion, key)
	return url, nil
}

// getPresignedURL generates a presigned URL for S3 object access
func getPresignedURL(bucketName, key string, expiration time.Duration) (string, error) {
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsRegion := os.Getenv("AWS_REGION")
	
	if awsRegion == "" {
		awsRegion = "us-east-1"
	}
	
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create AWS session: %v", err)
	}
	
	// Create S3 service client
	svc := s3.New(sess)
	
	// Create presigned URL request
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	
	// Generate presigned URL
	urlStr, err := req.Presign(expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}
	
	return urlStr, nil
}

// uploadToS3 uploads a file to S3 bucket (kept for backward compatibility)
func uploadToS3(filePath, bucketName, key string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()
	
	contentType := "video/mp4"
	if strings.HasSuffix(key, ".srt") {
		contentType = "text/plain"
	}
	
	return uploadToS3FromReader(file, bucketName, key, contentType)
}

func main() {
	// Load environment variables (optional in Docker, uses env vars directly)
	godotenv.Load("../.env")
	godotenv.Load(".env")
	
	apiKey := os.Getenv("ASSEMBLYAI_KEY")
	if apiKey == "" {
		log.Fatal("ASSEMBLYAI_KEY environment variable is required")
	}

	// Create necessary directories (minimal, only for static assets)
	os.MkdirAll("static", 0755)

	r := gin.Default()
	
	// Serve static files
	r.Static("/static", "./static")

	// Load HTML templates
	r.LoadHTMLGlob("templates/*")

	// GET / - Upload page
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", nil)
	})

	// GET /health - Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "captioning-backend",
		})
	})

	// GET /download/:filename - Proxy download from Remotion service
	r.GET("/download/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		remotionURL := os.Getenv("RENDER_REMOTION_URL")
		if remotionURL == "" {
			remotionURL = "http://localhost:3000"
		}
		
		// Proxy the download request to Remotion service
		resp, err := http.Get(remotionURL + "/download/" + filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Download failed"})
			return
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		
		// Set headers for file download
		c.Header("Content-Type", "video/mp4")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		
		// Stream the file
		io.Copy(c.Writer, resp.Body)
	})

	// POST /upload - Handle video upload and store in S3
	r.POST("/upload", func(c *gin.Context) {
		// Enforce max upload size (200MB)
		const maxUploadSize = 200 * 1024 * 1024 // 200MB
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)
		
		if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 200MB)"})
			return
		}

		file, header, err := c.Request.FormFile("video")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}
		defer file.Close()

		// Validate MIME type using first 512 bytes
		buffer := make([]byte, 512)
		_, err = file.Read(buffer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}
		file.Seek(0, 0) // Reset file pointer

		mimeType := http.DetectContentType(buffer)
		if mimeType != "video/mp4" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only MP4 files are allowed"})
			return
		}

		// Generate secure filename with UUID
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".mp4"
		}
		filename := uuid.New().String() + ext
		
		// Upload directly to S3
		bucketName := os.Getenv("S3_BUCKET")
		if bucketName == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "S3_BUCKET not configured"})
			return
		}
		
		s3Key := fmt.Sprintf("uploads/%s", filename)
		s3URL, err := uploadToS3FromReader(file, bucketName, s3Key, "video/mp4")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload to S3: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"fileUrl": s3URL,
			"s3Key": s3Key,
		})
	})

	// POST /get-presigned-url - Get presigned URL for preview
	r.POST("/get-presigned-url", func(c *gin.Context) {
		var req struct {
			S3Key string `json:"s3Key"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		bucketName := os.Getenv("S3_BUCKET")
		if bucketName == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "S3_BUCKET not configured"})
			return
		}
		// Generate presigned URL valid for 1 hour
		presignedURL, err := getPresignedURL(bucketName, req.S3Key, 1*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate presigned URL: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": presignedURL})
	})

	// POST /transcribe - Transcribe video using AssemblyAI
	r.POST("/transcribe", func(c *gin.Context) {
		var req struct {
			FileURL string `json:"fileUrl"`
			S3Key   string `json:"s3Key"`
		}
		
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Step 1: Generate presigned URL for AssemblyAI to access the video
		bucketName := os.Getenv("S3_BUCKET")
		if bucketName == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "S3_BUCKET not configured"})
			return
		}
		
		// Generate presigned URL valid for 1 hour
		presignedURL, err := getPresignedURL(bucketName, req.S3Key, 1*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate presigned URL: %v", err)})
			return
		}
		
		log.Printf("Generated presigned URL for transcription: %s", presignedURL)

		// Step 2: Request transcription using presigned URL
		transcriptID, err := requestTranscription(presignedURL, apiKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Transcription request failed: %v", err)})
			return
		}

		// Step 3: Poll for completion
		transcript, err := pollTranscription(transcriptID, apiKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Transcription failed: %v", err)})
			return
		}

		// Step 4: Convert to captions
		captions := convertToCaptions(transcript.Words)
		
		// Step 5: Generate SRT file and upload to S3
		srtContent := generateSRT(captions)
		
		srtKey := fmt.Sprintf("captions/%d.srt", time.Now().Unix())
		srtURL, err := uploadToS3FromReader(strings.NewReader(srtContent), bucketName, srtKey, "text/plain")
		if err != nil {
			log.Printf("Failed to upload SRT to S3: %v", err)
			// Continue anyway, captions are in response
		}

		c.JSON(http.StatusOK, gin.H{
			"captions": captions,
			"srtUrl":   srtURL,
		})
	})

	// POST /render-job - Trigger Remotion render and upload to S3
	r.POST("/render-job", func(c *gin.Context) {
		var req struct {
			VideoURL string    `json:"videoUrl"`
			Captions []Caption `json:"captions"`
			Style    string    `json:"style"`
			S3Key    string    `json:"s3Key"`
		}
		
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		remotionURL := os.Getenv("RENDER_REMOTION_URL")
		if remotionURL == "" {
			remotionURL = "http://localhost:3000"
		}

		bucketName := os.Getenv("S3_BUCKET")
		if bucketName == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "S3_BUCKET not configured"})
			return
		}

		// Generate presigned URL for Remotion to access the video (valid for 2 hours)
		var videoURLForRender string
		if req.S3Key != "" {
			presignedURL, err := getPresignedURL(bucketName, req.S3Key, 2*time.Hour)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate presigned URL: %v", err)})
				return
			}
			videoURLForRender = presignedURL
			log.Printf("Generated presigned URL for rendering: %s", presignedURL)
		} else {
			// Fallback to provided URL
			videoURLForRender = strings.ReplaceAll(req.VideoURL, "\\", "/")
		}
		
		renderReq := map[string]interface{}{
			"videoUrl": videoURLForRender,
			"captions": req.Captions,
			"style":    req.Style,
			"outPath":  fmt.Sprintf("out/video_%d.mp4", time.Now().Unix()),
		}

		jsonData, _ := json.Marshal(renderReq)
		
		// Create request with API key header
		httpReq, err := http.NewRequest("POST", remotionURL+"/render", bytes.NewBuffer(jsonData))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": "Failed to create render request",
			})
			return
		}
		
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", os.Getenv("RENDER_API_KEY"))
		
		client := &http.Client{Timeout: 10 * time.Minute}
		resp, err := client.Do(httpReq)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": "Remotion service unavailable",
			})
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		
		// If render was successful, upload to S3
		if success, ok := result["success"].(bool); ok && success {
			if outPath, ok := result["outPath"].(string); ok {
				// Download the file from Remotion service
				filename := filepath.Base(outPath)
				downloadURL := remotionURL + "/download/" + filename
				
				resp, err := http.Get(downloadURL)
				if err != nil {
					log.Printf("Failed to download from Remotion: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"error": "Failed to download rendered video",
					})
					return
				}
				defer resp.Body.Close()
				
				if resp.StatusCode == http.StatusOK {
					// Upload directly to S3 from stream
					s3Key := fmt.Sprintf("output/%s", filename)
					s3URL, err := uploadToS3FromReader(resp.Body, bucketName, s3Key, "video/mp4")
					if err != nil {
						log.Printf("Failed to upload to S3: %v", err)
						c.JSON(http.StatusInternalServerError, gin.H{
							"success": false,
							"error": "Failed to upload to S3",
						})
						return
					}
					
					log.Printf("Video uploaded to S3: %s", s3URL)
					
					// Generate presigned URL for download (valid for 24 hours)
					presignedDownloadURL, err := getPresignedURL(bucketName, s3Key, 24*time.Hour)
					if err != nil {
						log.Printf("Failed to generate presigned download URL: %v", err)
						// Fallback to regular URL
						presignedDownloadURL = s3URL
					}
					
					c.JSON(http.StatusOK, gin.H{
						"success": true,
						"s3_url": presignedDownloadURL,
						"message": "Video rendered and uploaded to S3 successfully",
					})
					return
				} else {
					log.Printf("Failed to download from Remotion: status %d", resp.StatusCode)
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"error": "Rendered video not found",
					})
					return
				}
			}
		}
		
		c.JSON(http.StatusOK, result)
	})

	log.Println("Server starting on :7070")
	r.Run(":7070")
}

// uploadToAssemblyAI uploads a local file to AssemblyAI
func uploadToAssemblyAI(filePath, apiKey string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	req, err := http.NewRequest("POST", "https://api.assemblyai.com/v2/upload", file)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("authorization", apiKey)
	req.Header.Set("content-type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var uploadResp AssemblyAIUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", err
	}

	return uploadResp.UploadURL, nil
}

// requestTranscription starts a transcription job
func requestTranscription(audioURL, apiKey string) (string, error) {
	reqBody := AssemblyAITranscriptRequest{AudioURL: audioURL}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.assemblyai.com/v2/transcript", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("authorization", apiKey)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var transcriptResp AssemblyAITranscriptResponse
	if err := json.NewDecoder(resp.Body).Decode(&transcriptResp); err != nil {
		return "", err
	}

	return transcriptResp.ID, nil
}

// pollTranscription polls until transcription is complete
// FIX #3: Add timeout, max attempts, and exponential backoff
func pollTranscription(transcriptID, apiKey string) (*AssemblyAITranscriptResponse, error) {
	url := fmt.Sprintf("https://api.assemblyai.com/v2/transcript/%s", transcriptID)
	
	// Create context with 10-minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	maxAttempts := 40
	backoff := 1 * time.Second
	
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check if context is done
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("transcription timed out after 10 minutes")
		default:
		}
		
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("authorization", apiKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		var transcript AssemblyAITranscriptResponse
		json.NewDecoder(resp.Body).Decode(&transcript)
		resp.Body.Close()

		if transcript.Status == "completed" {
			return &transcript, nil
		} else if transcript.Status == "error" {
			return nil, fmt.Errorf("transcription failed")
		}

		// Exponential backoff: 1s → 2s → 4s → 8s → 16s → max 30s
		time.Sleep(backoff)
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
	
	return nil, fmt.Errorf("transcription timed out after %d attempts", maxAttempts)
}

// convertToCaptions converts AssemblyAI words to caption segments
func convertToCaptions(words []struct {
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}) []Caption {
	var captions []Caption
	var currentCaption Caption
	wordCount := 0
	maxWordsPerCaption := 8

	for _, word := range words {
		if wordCount == 0 {
			currentCaption = Caption{
				Start: float64(word.Start) / 1000.0,
				Text:  word.Text,
			}
		} else {
			currentCaption.Text += " " + word.Text
		}
		
		currentCaption.End = float64(word.End) / 1000.0
		wordCount++

		if wordCount >= maxWordsPerCaption {
			captions = append(captions, currentCaption)
			wordCount = 0
		}
	}

	if wordCount > 0 {
		captions = append(captions, currentCaption)
	}

	return captions
}

// generateSRT creates SRT format from captions
func generateSRT(captions []Caption) string {
	var srt strings.Builder
	
	for i, caption := range captions {
		srt.WriteString(fmt.Sprintf("%d\n", i+1))
		srt.WriteString(fmt.Sprintf("%s --> %s\n", formatSRTTime(caption.Start), formatSRTTime(caption.End)))
		srt.WriteString(fmt.Sprintf("%s\n\n", caption.Text))
	}
	
	return srt.String()
}

// formatSRTTime converts seconds to SRT time format (HH:MM:SS,ms)
func formatSRTTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)
	
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

// generateCLICommand creates a CLI fallback command
func generateCLICommand(videoURL string, captions []Caption, style string) string {
	captionsJSON, _ := json.Marshal(captions)
	propsJSON := fmt.Sprintf(`{"videoUrl":"%s","captions":%s,"style":"%s"}`, videoURL, captionsJSON, style)
	
	return fmt.Sprintf(`cd remotion-app && npx remotion render src/index.tsx CaptionedVideo out/final.mp4 --props '%s'`, propsJSON)
}
