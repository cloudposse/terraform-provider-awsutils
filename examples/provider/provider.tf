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

# Export default VPN client config
data "awsutils_ec2_client_vpn_export_client_config" "default" {
    client_vpn_endpoint_id = aws_ec2_client_vpn_endpoint.default.id
}
