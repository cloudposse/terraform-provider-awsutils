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

## TODO: Update this once we have some resources
# Create a VPC
resource "awsutils_vpc" "example" {
  cidr_block = "10.0.0.0/16"
}
