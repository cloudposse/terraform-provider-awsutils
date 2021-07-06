terraform {
  required_providers {
    awsutils = {
      source  = "cloudposse/awsutils"
      version = "~> 1.0"
    }
  }
}

# Configure the AWS Provider
provider "awsutils" {
  region = "us-east-1"
}

# Delete default VPC
resource "awsutils_default_vpc_deletion" "example" {
}
