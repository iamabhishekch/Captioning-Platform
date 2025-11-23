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
