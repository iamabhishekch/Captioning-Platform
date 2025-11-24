output "sqs_queue_url" {
  description = "SQS queue URL for render jobs"
  value       = aws_sqs_queue.render_queue.url
}

output "dynamodb_table_name" {
  description = "DynamoDB table name for job status"
  value       = aws_dynamodb_table.render_jobs.name
}

output "lambda_function_name" {
  description = "Lambda function name"
  value       = aws_lambda_function.render_worker.function_name
}

output "lambda_function_arn" {
  description = "Lambda function ARN"
  value       = aws_lambda_function.render_worker.arn
}

output "ecr_repository_url" {
  description = "ECR repository URL for Remotion service"
  value       = aws_ecr_repository.remotion.repository_url
}

output "remotion_url" {
  description = "Remotion service URL"
  value       = "http://${aws_lb.remotion.dns_name}"
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.main.name
}

output "ecs_service_name" {
  description = "ECS service name"
  value       = aws_ecs_service.remotion.name
}


output "backend_url" {
  description = "Backend service URL"
  value       = "http://${aws_lb.remotion.dns_name}:7070"
}
