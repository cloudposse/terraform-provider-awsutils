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

resource "awsutils_guardduty_organization_settings" "default" {
  member_accounts = ["111111111111", "22222222222"]
  detector_id     = "42bd3eab69b96663418094bb59397d1f"
}
