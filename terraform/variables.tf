variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "video-captioning"
}

variable "s3_bucket" {
  description = "S3 bucket for video storage"
  type        = string
  default     = "go-video-project-685a78d468a5w4e65"
}

variable "assemblyai_key" {
  description = "AssemblyAI API key"
  type        = string
  sensitive   = true
}

variable "render_api_key" {
  description = "Remotion render API key"
  type        = string
  default     = "secure_key_12345"
  sensitive   = true
}
