# SQS Queue for render jobs
resource "aws_sqs_queue" "render_queue" {
  name                       = "${var.project_name}-render-queue"
  visibility_timeout_seconds = 900  # 15 minutes for Lambda to process
  message_retention_seconds  = 86400 # 24 hours
  receive_wait_time_seconds  = 20   # Long polling

  tags = {
    Name        = "${var.project_name}-render-queue"
    Environment = "production"
  }
}

# DynamoDB table for job status
resource "aws_dynamodb_table" "render_jobs" {
  name           = "${var.project_name}-jobs"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "jobId"

  attribute {
    name = "jobId"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  global_secondary_index {
    name            = "StatusIndex"
    hash_key        = "status"
    projection_type = "ALL"
  }

  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }

  tags = {
    Name        = "${var.project_name}-jobs"
    Environment = "production"
  }
}

# IAM role for Lambda
resource "aws_iam_role" "lambda_render" {
  name = "${var.project_name}-lambda-render"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

# IAM policy for Lambda
resource "aws_iam_role_policy" "lambda_render_policy" {
  name = "${var.project_name}-lambda-policy"
  role = aws_iam_role.lambda_render.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes"
        ]
        Resource = aws_sqs_queue.render_queue.arn
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:UpdateItem"
        ]
        Resource = aws_dynamodb_table.render_jobs.arn
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject"
        ]
        Resource = "arn:aws:s3:::${var.s3_bucket}/*"
      }
    ]
  })
}

# Lambda function (placeholder - will be deployed separately)
resource "aws_lambda_function" "render_worker" {
  filename      = "lambda_placeholder.zip"
  function_name = "${var.project_name}-render-worker"
  role          = aws_iam_role.lambda_render.arn
  handler       = "index.handler"
  runtime       = "nodejs18.x"
  timeout       = 900 # 15 minutes
  memory_size   = 3008 # Maximum memory for faster rendering

  environment {
    variables = {
      DYNAMODB_TABLE = aws_dynamodb_table.render_jobs.name
      S3_BUCKET      = var.s3_bucket
      RENDER_API_KEY = var.render_api_key
    }
  }

  lifecycle {
    ignore_changes = [filename, source_code_hash]
  }
}

# Lambda SQS trigger
resource "aws_lambda_event_source_mapping" "sqs_trigger" {
  event_source_arn = aws_sqs_queue.render_queue.arn
  function_name    = aws_lambda_function.render_worker.arn
  batch_size       = 1
}
