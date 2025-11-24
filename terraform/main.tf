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
      },
      {
        Effect = "Allow"
        Action = [
          "ec2:CreateNetworkInterface",
          "ec2:DescribeNetworkInterfaces",
          "ec2:DeleteNetworkInterface"
        ]
        Resource = "*"
      }
    ]
  })
}

# ECR Repository for Remotion service
resource "aws_ecr_repository" "remotion" {
  name                 = "${var.project_name}-remotion"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }

  tags = {
    Name        = "${var.project_name}-remotion"
    Environment = "production"
  }
}

# VPC for ECS
resource "aws_default_vpc" "default" {}

resource "aws_default_subnet" "default_az1" {
  availability_zone = "${var.aws_region}a"
}

resource "aws_default_subnet" "default_az2" {
  availability_zone = "${var.aws_region}b"
}

# Security Group for Remotion service
resource "aws_security_group" "remotion" {
  name        = "${var.project_name}-remotion-sg"
  description = "Security group for Remotion ECS service"
  vpc_id      = aws_default_vpc.default.id

  ingress {
    from_port   = 3000
    to_port     = 3000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-remotion-sg"
  }
}

# ECS Cluster
resource "aws_ecs_cluster" "main" {
  name = "${var.project_name}-cluster"

  tags = {
    Name        = "${var.project_name}-cluster"
    Environment = "production"
  }
}

# ECS Task Definition for Remotion
resource "aws_ecs_task_definition" "remotion" {
  family                   = "${var.project_name}-remotion"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "2048"  # 2 vCPU
  memory                   = "4096"  # 4 GB
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([{
    name  = "remotion"
    image = "${aws_ecr_repository.remotion.repository_url}:latest"
    
    portMappings = [{
      containerPort = 3000
      protocol      = "tcp"
    }]

    environment = [
      {
        name  = "RENDER_API_KEY"
        value = var.render_api_key
      }
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = "/ecs/${var.project_name}-remotion"
        "awslogs-region"        = var.aws_region
        "awslogs-stream-prefix" = "ecs"
        "awslogs-create-group"  = "true"
      }
    }
  }])
}

# Service Discovery Namespace
resource "aws_service_discovery_private_dns_namespace" "main" {
  name        = "local"
  description = "Private DNS namespace for service discovery"
  vpc         = aws_default_vpc.default.id
}

# Service Discovery Service
resource "aws_service_discovery_service" "remotion" {
  name = "remotion"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.main.id

    dns_records {
      ttl  = 10
      type = "A"
    }
  }

  health_check_custom_config {
    failure_threshold = 1
  }
}

# ECS Service with Fargate Spot
resource "aws_ecs_service" "remotion" {
  name            = "${var.project_name}-remotion"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.remotion.arn
  desired_count   = 1

  capacity_provider_strategy {
    capacity_provider = "FARGATE_SPOT"
    weight            = 100
    base              = 0
  }

  network_configuration {
    subnets          = [aws_default_subnet.default_az1.id, aws_default_subnet.default_az2.id]
    security_groups  = [aws_security_group.remotion.id]
    assign_public_ip = true
  }

  service_registries {
    registry_arn = aws_service_discovery_service.remotion.arn
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.remotion.arn
    container_name   = "remotion"
    container_port   = 3000
  }

  depends_on = [aws_lb_listener.remotion]
}

# Application Load Balancer
resource "aws_lb" "remotion" {
  name               = "${var.project_name}-remotion-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.remotion.id]
  subnets            = [aws_default_subnet.default_az1.id, aws_default_subnet.default_az2.id]

  tags = {
    Name = "${var.project_name}-remotion-alb"
  }
}

resource "aws_lb_target_group" "remotion" {
  name        = "${var.project_name}-remotion-tg"
  port        = 3000
  protocol    = "HTTP"
  vpc_id      = aws_default_vpc.default.id
  target_type = "ip"

  health_check {
    path                = "/health"
    healthy_threshold   = 2
    unhealthy_threshold = 10
    timeout             = 60
    interval            = 120
    matcher             = "200,404"
  }
}

resource "aws_lb_listener" "remotion" {
  load_balancer_arn = aws_lb.remotion.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.remotion.arn
  }
}

# IAM Roles for ECS
resource "aws_iam_role" "ecs_execution" {
  name = "${var.project_name}-ecs-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_execution" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy" "ecs_execution_logs" {
  name = "${var.project_name}-ecs-logs"
  role = aws_iam_role.ecs_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ]
      Resource = "*"
    }]
  })
}

resource "aws_iam_role" "ecs_task" {
  name = "${var.project_name}-ecs-task"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy" "ecs_task_s3" {
  name = "${var.project_name}-ecs-task-s3"
  role = aws_iam_role.ecs_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "s3:GetObject",
        "s3:PutObject"
      ]
      Resource = "arn:aws:s3:::${var.s3_bucket}/*"
    }]
  })
}

# Lambda function
resource "aws_lambda_function" "render_worker" {
  filename      = "lambda_placeholder.zip"
  function_name = "${var.project_name}-render-worker"
  role          = aws_iam_role.lambda_render.arn
  handler       = "index.handler"
  runtime       = "nodejs18.x"
  timeout       = 900
  memory_size   = 512

  vpc_config {
    subnet_ids         = [aws_default_subnet.default_az1.id, aws_default_subnet.default_az2.id]
    security_group_ids = [aws_security_group.lambda.id]
  }

  environment {
    variables = {
      DYNAMODB_TABLE = aws_dynamodb_table.render_jobs.name
      S3_BUCKET      = var.s3_bucket
      REMOTION_URL   = "http://remotion.local:3000"
      RENDER_API_KEY = var.render_api_key
    }
  }

  lifecycle {
    ignore_changes = [filename, source_code_hash]
  }
}

# Security Group for Lambda
resource "aws_security_group" "lambda" {
  name        = "${var.project_name}-lambda-sg"
  description = "Security group for Lambda function"
  vpc_id      = aws_default_vpc.default.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-lambda-sg"
  }
}

# Lambda SQS trigger
resource "aws_lambda_event_source_mapping" "sqs_trigger" {
  event_source_arn = aws_sqs_queue.render_queue.arn
  function_name    = aws_lambda_function.render_worker.arn
  batch_size       = 1
}


# VPC Endpoints for Lambda to access AWS services without NAT Gateway
resource "aws_vpc_endpoint" "dynamodb" {
  vpc_id            = aws_default_vpc.default.id
  service_name      = "com.amazonaws.${var.aws_region}.dynamodb"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = [aws_default_vpc.default.default_route_table_id]

  tags = {
    Name = "${var.project_name}-dynamodb-endpoint"
  }
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id            = aws_default_vpc.default.id
  service_name      = "com.amazonaws.${var.aws_region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = [aws_default_vpc.default.default_route_table_id]

  tags = {
    Name = "${var.project_name}-s3-endpoint"
  }
}


# ECS Task Definition for Backend
resource "aws_ecs_task_definition" "backend" {
  family                   = "${var.project_name}-backend"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.backend_task.arn

  container_definitions = jsonencode([{
    name  = "backend"
    image = "929910138721.dkr.ecr.us-east-1.amazonaws.com/video-captioning-backend:latest"
    
    portMappings = [{
      containerPort = 7070
      protocol      = "tcp"
    }]

    environment = [
      {
        name  = "ASSEMBLYAI_KEY"
        value = var.assemblyai_key
      },
      {
        name  = "S3_BUCKET"
        value = var.s3_bucket
      },
      {
        name  = "AWS_REGION"
        value = var.aws_region
      },
      {
        name  = "SQS_QUEUE_URL"
        value = aws_sqs_queue.render_queue.url
      },
      {
        name  = "DYNAMODB_TABLE"
        value = aws_dynamodb_table.render_jobs.name
      }
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = "/ecs/${var.project_name}-backend"
        "awslogs-region"        = var.aws_region
        "awslogs-stream-prefix" = "ecs"
        "awslogs-create-group"  = "true"
      }
    }
  }])
}

# IAM Role for Backend Task
resource "aws_iam_role" "backend_task" {
  name = "${var.project_name}-backend-task"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy" "backend_task_policy" {
  name = "${var.project_name}-backend-task-policy"
  role = aws_iam_role.backend_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "arn:aws:s3:::${var.s3_bucket}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage",
          "sqs:GetQueueUrl"
        ]
        Resource = aws_sqs_queue.render_queue.arn
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:Query"
        ]
        Resource = aws_dynamodb_table.render_jobs.arn
      }
    ]
  })
}

# Security Group for Backend
resource "aws_security_group" "backend" {
  name        = "${var.project_name}-backend-sg"
  description = "Security group for Backend ECS service"
  vpc_id      = aws_default_vpc.default.id

  ingress {
    from_port   = 7070
    to_port     = 7070
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-backend-sg"
  }
}

# ECS Service for Backend
resource "aws_ecs_service" "backend" {
  name            = "${var.project_name}-backend"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.backend.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = [aws_default_subnet.default_az1.id, aws_default_subnet.default_az2.id]
    security_groups  = [aws_security_group.backend.id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.backend.arn
    container_name   = "backend"
    container_port   = 7070
  }

  depends_on = [aws_lb_listener.backend]
}

# Target Group for Backend
resource "aws_lb_target_group" "backend" {
  name        = "${var.project_name}-backend-tg"
  port        = 7070
  protocol    = "HTTP"
  vpc_id      = aws_default_vpc.default.id
  target_type = "ip"

  health_check {
    path                = "/health"
    healthy_threshold   = 2
    unhealthy_threshold = 10
    timeout             = 60
    interval            = 120
    matcher             = "200"
  }
}

# ALB Listener for Backend
resource "aws_lb_listener" "backend" {
  load_balancer_arn = aws_lb.remotion.arn
  port              = "7070"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.backend.arn
  }
}
