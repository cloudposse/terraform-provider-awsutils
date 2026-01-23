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

# Delete the default VPC in the provider's region
resource "awsutils_default_vpc_deletion" "default" {
}

# Delete the default VPC in a specific region
resource "awsutils_default_vpc_deletion" "eu_west_1" {
  region = "eu-west-1"
}

# Delete default VPCs in multiple regions using for_each
variable "regions" {
  description = "List of AWS regions to delete default VPCs from"
  type        = list(string)
  default = [
    "us-east-1",
    "us-east-2",
    "us-west-1",
    "us-west-2",
    "eu-west-1",
    "eu-west-2",
    "eu-central-1",
    "ap-southeast-1",
    "ap-southeast-2",
    "ap-northeast-1",
  ]
}

resource "awsutils_default_vpc_deletion" "all_regions" {
  for_each = toset(var.regions)
  region   = each.value
}
