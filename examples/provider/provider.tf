terraform {
  required_providers {
    rmq = {
      source = "nbr23/rmq"
    }
  }
}

provider "rmq" {
  uri = "amqp://guest:guest@localhost:5672/"
}
