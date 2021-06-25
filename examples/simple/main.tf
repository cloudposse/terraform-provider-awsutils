terraform {
  required_version = ">= 0.12"

  required_providers {
    awsutils = {
      source = "example.com/cloudposse/awsutils"
      #version = "~> 1.0"
    }
  }
}

provider "awsutils" {
  region = "us-east-1"
}

# Create a VPC to launch our instances into
resource "awsutils_vpc" "default" {
  cidr_block = "10.0.0.0/16"
}
