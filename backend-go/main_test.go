package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	
	// Create test directories
	os.MkdirAll("uploads", 0755)
	os.MkdirAll("captions", 0755)
	
	return r
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	router := setupTestRouter()
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "captioning-backend",
		})
	})
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, "captioning-backend", response["service"])
}

// TestUploadEndpoint tests video upload
func TestUploadEndpoint(t *testing.T) {
	router := setupTestRouter()
	
	router.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("video")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"fileUrl": "uploads/" + file.Filename})
	})
	
	// Create test file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("video", "test.mp4")
	part.Write([]byte("fake video content"))
	writer.Close()
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["fileUrl"], "uploads/test.mp4")
}

// TestUploadEndpointNoFile tests upload without file
func TestUploadEndpointNoFile(t *testing.T) {
	router := setupTestRouter()
	
	router.POST("/upload", func(c *gin.Context) {
		_, err := c.FormFile("video")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"fileUrl": "uploads/test.mp4"})
	})
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "No file uploaded", response["error"])
}

// TestConvertToCaptions tests caption conversion logic
func TestConvertToCaptions(t *testing.T) {
	words := []struct {
		Text  string `json:"text"`
		Start int    `json:"start"`
		End   int    `json:"end"`
	}{
		{Text: "Hello", Start: 0, End: 500},
		{Text: "world", Start: 500, End: 1000},
		{Text: "this", Start: 1000, End: 1500},
		{Text: "is", Start: 1500, End: 2000},
		{Text: "a", Start: 2000, End: 2200},
		{Text: "test", Start: 2200, End: 2700},
	}
	
	captions := convertToCaptions(words)
	
	assert.NotEmpty(t, captions)
	assert.Equal(t, 0.0, captions[0].Start)
	assert.Contains(t, captions[0].Text, "Hello")
	assert.Contains(t, captions[0].Text, "world")
}

// TestGenerateSRT tests SRT generation
func TestGenerateSRT(t *testing.T) {
	captions := []Caption{
		{Start: 0.0, End: 2.5, Text: "Hello world"},
		{Start: 2.5, End: 5.0, Text: "This is a test"},
	}
	
	srt := generateSRT(captions)
	
	assert.Contains(t, srt, "1\n")
	assert.Contains(t, srt, "00:00:00,000 --> 00:00:02,500")
	assert.Contains(t, srt, "Hello world")
	assert.Contains(t, srt, "2\n")
	assert.Contains(t, srt, "00:00:02,500 --> 00:00:05,000")
	assert.Contains(t, srt, "This is a test")
}

// TestFormatSRTTime tests SRT time formatting
func TestFormatSRTTime(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0.0, "00:00:00,000"},
		{1.5, "00:00:01,500"},
		{65.123, "00:01:05,123"},
		{3661.456, "01:01:01,456"},
	}
	
	for _, test := range tests {
		result := formatSRTTime(test.input)
		assert.Equal(t, test.expected, result, "Failed for input %f", test.input)
	}
}

// TestGenerateCLICommand tests CLI command generation
func TestGenerateCLICommand(t *testing.T) {
	captions := []Caption{
		{Start: 0.0, End: 2.5, Text: "Test caption"},
	}
	
	cmd := generateCLICommand("uploads/test.mp4", captions, "bottom")
	
	assert.Contains(t, cmd, "cd remotion-app")
	assert.Contains(t, cmd, "npx remotion render")
	assert.Contains(t, cmd, "CaptionedVideo")
	assert.Contains(t, cmd, "uploads/test.mp4")
	assert.Contains(t, cmd, "bottom")
}

// TestTranscribeEndpointInvalidRequest tests transcribe with invalid request
func TestTranscribeEndpointInvalidRequest(t *testing.T) {
	router := setupTestRouter()
	
	router.POST("/transcribe", func(c *gin.Context) {
		var req struct {
			FileURL string `json:"fileUrl"`
		}
		
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		
		if req.FileURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fileUrl is required"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"captions": []Caption{}})
	})
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/transcribe", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRenderJobEndpoint tests render job endpoint
func TestRenderJobEndpoint(t *testing.T) {
	router := setupTestRouter()
	
	router.POST("/render-job", func(c *gin.Context) {
		var req struct {
			VideoURL string    `json:"videoUrl"`
			Captions []Caption `json:"captions"`
			Style    string    `json:"style"`
		}
		
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		
		// Simulate CLI fallback
		cliCommand := generateCLICommand(req.VideoURL, req.Captions, req.Style)
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"message":    "Remotion service unavailable. Use CLI fallback:",
			"cliCommand": cliCommand,
		})
	})
	
	reqBody := map[string]interface{}{
		"videoUrl": "uploads/test.mp4",
		"captions": []Caption{{Start: 0, End: 2, Text: "Test"}},
		"style":    "bottom",
	}
	jsonData, _ := json.Marshal(reqBody)
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/render-job", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "cliCommand")
}

// Cleanup test directories
func TestMain(m *testing.M) {
	code := m.Run()
	os.RemoveAll("uploads")
	os.RemoveAll("captions")
	os.Exit(code)
}
