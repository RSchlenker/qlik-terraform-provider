// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccAppResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAppResourceConfig("test-app"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-app"),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test application for acceptance testing"),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("created_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("modified_date"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "qlik_app.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccAppResourceConfigUpdated("updated-test-app"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("updated-test-app"),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated test application for acceptance testing"),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Remove description testing
			{
				Config: testAccAppResourceConfigWithoutDescription("test-app-no-description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-app-no-description"),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("description"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Second apply should be no-op
			{
				Config: testAccAppResourceConfigWithoutDescription("test-app-no-description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-app-no-description"),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("description"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"qlik_app.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Verify owner_id is computed and cannot be configured
			{
				Config:      testAccAppResourceConfigWithOwnerId("test-app-owner-id"),
				ExpectError: regexp.MustCompile(`owner_id.*computed.*cannot be configured`),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAppResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "qlik_app" "test" {
  name        = %[1]q
  description = "Test application for acceptance testing"
}
`, name)
}

func testAccAppResourceConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "qlik_app" "test" {
  name        = %[1]q
  description = "Updated test application for acceptance testing"
}
`, name)
}

func testAccAppResourceConfigWithoutDescription(name string) string {
	return fmt.Sprintf(`
resource "qlik_app" "test" {
  name = %[1]q
}
`, name)
}

func testAccAppResourceConfigWithOwnerId(name string) string {
	return fmt.Sprintf(`
resource "qlik_app" "test" {
  name     = %[1]q
  owner_id = "some-user-id"
}
`, name)
}