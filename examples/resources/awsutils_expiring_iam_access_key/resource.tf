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

resource "aws_iam_user" "test" {
  name = "test"
  path = "/test/"
}

resource "awsutils_expiring_iam_access_key" "test" {
  user    = aws_iam_user.test.name
  max_age = 60 * 60 * 24 * 30 # 30 days
}

output "id" {
  value = awsutils_expiring_iam_access_key.test.id
}
