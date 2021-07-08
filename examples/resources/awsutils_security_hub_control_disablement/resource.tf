terraform {
  required_providers {
    awsutils = {
      source = "cloudposse/awsutils"
      # For local development,
      # install the provider on local computer by running `make install` from the root of the repo, and uncomment the 
      # version below
      # version = "9999.99.99"
    }
  }
}

provider "awsutils" {
  region = "us-east-1"
}

resource "awsutils_security_hub_control_disablement" "default" {
  control_arn = "arn:aws:securityhub:${data.aws_region.this.name}:${data.aws_caller_identity.this.account_id}:control/cis-aws-foundations-benchmark/v/1.2.0/1.1"
  reason      = "Global Resources are not evaluated in this region"
}

data "aws_region" "this" {}
data "aws_caller_identity" "this" {}
