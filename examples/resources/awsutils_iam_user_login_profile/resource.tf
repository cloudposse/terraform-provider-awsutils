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

resource "awsutils_iam_user_login_profile" "default" {

  user                    = "j.kowalski"
  password                = "zaq1@WSX"
  password_reset_required = true
}
