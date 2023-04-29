terraform {
  required_providers {
    awsutils = {
      source = "cloudposse/awsutils"
      # For local development,
      # install the provider on local computer by running `make install` from the root of the repo,
      # and uncomment the version below
      version = "9999.99.99"
    }
  }
}

# Configure the AWS Provider
provider "awsutils" {
  region = "us-east-1"
}

data "awsutils_caller_identity" "default" {
}

output "account_id" {
  value = data.awsutils_caller_identity.default.account_id
}

output "caller_arn" {
  value = data.awsutils_caller_identity.default.arn
}

output "caller_user" {
  value = data.awsutils_caller_identity.default.user_id
}

output "eks_role_arn" {
  value = data.awsutils_caller_identity.default.eks_role_arn
}
