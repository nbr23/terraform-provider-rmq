terraform {
  required_providers {
    rmq = {
      source = "registry.terraform.io/nbr23/rmq"
    }
  }
}

provider "rmq" {
  uri = "amqp://guest:guest@localhost:5672/"
}


data "rmq_message" "msg" {
  routing_key       = "#"
  exchange          = "test-exchange"
  ack               = true
  message_count     = 1
  queue_name_prefix = "temp-queue-"
}

output "message" {
  value = data.rmq_message.msg
}
