resource "aws_dynamodb_table" "tags" {
  name           = "l0-${var.name}-tags"
  read_capacity  = 5
  write_capacity = 5
  hash_key       = "EntityType"
  range_key      = "EntityID"
  tags           = "${var.tags}"

  attribute {
    name = "EntityType"
    type = "S"
  }

  attribute {
    name = "EntityID"
    type = "S"
  }
}

resource "aws_dynamodb_table" "jobs" {
  name           = "l0-${var.name}-jobs"
  read_capacity  = 5
  write_capacity = 10
  hash_key       = "JobID"

  attribute {
    name = "JobID"
    type = "S"
  }
}

resource "aws_appautoscaling_target" "tags_table_read_target" {
  max_capacity       = 250
  min_capacity       = 5
  resource_id        = "table/${aws_dynamodb_table.tags.name}"
  scalable_dimension = "dynamodb:table:ReadCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "tags_table_read_policy" {
  name               = "l0-${var.name}-tags-table-read-capacity-utilization:${aws_appautoscaling_target.tags_table_read_target.resource_id}"
  policy_type        = "TargetTrackingScaling"
  resource_id        = "${aws_appautoscaling_target.tags_table_read_target.resource_id}"
  scalable_dimension = "${aws_appautoscaling_target.tags_table_read_target.scalable_dimension}"
  service_namespace  = "${aws_appautoscaling_target.tags_table_read_target.service_namespace}"

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBReadCapacityUtilization"
    }

    target_value = 70
  }
}

resource "aws_appautoscaling_target" "jobs_table_read_target" {
  max_capacity       = 250
  min_capacity       = 5
  resource_id        = "table/${aws_dynamodb_table.jobs.name}"
  scalable_dimension = "dynamodb:table:ReadCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "jobs_table_read_policy" {
  name               = "l0-${var.name}-jobs-table-read-capacity-utilization:${aws_appautoscaling_target.jobs_table_read_target.resource_id}"
  policy_type        = "TargetTrackingScaling"
  resource_id        = "${aws_appautoscaling_target.jobs_table_read_target.resource_id}"
  scalable_dimension = "${aws_appautoscaling_target.jobs_table_read_target.scalable_dimension}"
  service_namespace  = "${aws_appautoscaling_target.jobs_table_read_target.service_namespace}"

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBReadCapacityUtilization"
    }

    target_value = 70
  }
}
