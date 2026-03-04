# Qlik Terraform Provider

A [Terraform](https://www.terraform.io) provider for managing [Qlik Cloud](https://www.qlik.com/us/products/qlik-cloud) resources.

Built with the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24

## Building The Provider

Clone the repository and build the provider binary:

```shell
git clone https://github.com/rschlenker/qlik-terraform-provider
cd qlik-terraform-provider
go build -o ~/go/bin/terraform-provider-qlik .
```

## Using the Provider Locally

Configure `~/.terraformrc` to use the local binary:

```hcl
provider_installation {
  dev_overrides {
    "rschlenker/qlik" = "/Users/<you>/go/bin"
  }
  direct {}
}
```

Then reference the provider in your Terraform configuration:

```hcl
terraform {
  required_providers {
    qlik = {
      source = "rschlenker/qlik"
    }
  }
}

provider "qlik" {
  tenant_id = "your-tenant"
  region    = "de"
  api_key   = var.qlik_api_key
}

```

With `dev_overrides` active, no `terraform init` is needed for the provider — run `terraform plan` or `terraform apply` directly.

## Developing the Provider

To add a new dependency:

```shell
go get github.com/author/dependency
go mod tidy
```

To regenerate the Qlik API SDK from the OpenAPI spec:

```shell
make generate
```

To run acceptance tests (creates real resources):

```shell
make testacc
```
