terraform {
  required_providers {
    awsutils = {
      source = "cloudposse/awsutils"
      # For local development,
      # install the provider on local computer by running `make install` from the root of the repo,
      # and uncomment the version below
      # version = "9999.99.99"
    }
  }
}

# Configure the AWS Provider
provider "awsutils" {
  region = "us-east-1"
}

data "awsutils_ec2_client_vpn_export_client_config" "default" {
  id = "test"
}

output "client_configuration" {
  value = data.awsutils_ec2_client_vpn_export_client_config.default.client_configuration
}