## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `atmos testacc`.

_Note:_ Acceptance tests create real resources, and often cost money to run.

```sh
$ atmos testacc
```

### Testing Locally

You can test the provider locally by using the [provider_installation](https://www.terraform.io/docs/cli/config/config-file.html#provider-installation) functionality.

For testing this provider, you can edit your `~/.terraformrc` file with the following:

```hcl
provider_installation {
  dev_overrides  {
    "cloudposse/awsutils" = "/path/to/your/code/github.com/cloudposse/terraform-provider-awsutils/"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

With that in place, you can build the provider (see above) and add a provider block:

```hcl
required_providers {
    awsutils = {
      source = "cloudposse/awsutils"
    }
  }
```

Then run `terraform init`, `terraform plan` and `terraform apply` as normal.

```sh
$ terraform init
Initializing the backend...

Initializing provider plugins...
- Finding latest version of cloudposse/awsutils...

Warning: Provider development overrides are in effect

The following provider development overrides are set in the CLI configuration:
 - cloudposse/awsutils in /path/to/your/code/github.com/cloudposse/terraform-provider-awsutils

The behavior may therefore not match any released version of the provider and
applying changes may cause the state to become incompatible with published
releases.
```

```sh
terraform apply

Warning: Provider development overrides are in effect

The following provider development overrides are set in the CLI configuration:
 - cloudposse/awsutils in /Users/matt/code/src/github.com/cloudposse/terraform-provider-awsutils

The behavior may therefore not match any released version of the provider and
applying changes may cause the state to become incompatible with published
releases.


An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:

Terraform will perform the following actions:

Plan: 1 to add, 0 to change, 0 to destroy.
```
