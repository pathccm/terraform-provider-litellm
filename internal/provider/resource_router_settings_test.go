package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRouterSettingsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRouterSettingsConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("litellm_router_settings.test", "id", "router_settings"),
					resource.TestCheckResourceAttr("litellm_router_settings.test", "routing_strategy", "least-busy"),
					resource.TestCheckResourceAttr("litellm_router_settings.test", "num_retries", "2"),
					resource.TestCheckResourceAttr("litellm_router_settings.test", "enable_pre_call_checks", "true"),
				),
			},
			{
				Config: testAccRouterSettingsConfig_updated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("litellm_router_settings.test", "routing_strategy", "latency-based-routing"),
					resource.TestCheckResourceAttr("litellm_router_settings.test", "num_retries", "3"),
				),
			},
		},
	})
}

func testAccRouterSettingsConfig_basic() string {
	return `
resource "litellm_router_settings" "test" {
  routing_strategy       = "least-busy"
  num_retries            = 2
  timeout                = 30
  cooldown_time          = 60
  allowed_fails          = 3
  enable_pre_call_checks = true
}
`
}

func testAccRouterSettingsConfig_updated() string {
	return `
resource "litellm_router_settings" "test" {
  routing_strategy       = "latency-based-routing"
  num_retries            = 3
  timeout                = 60
  cooldown_time          = 120
  allowed_fails          = 5
  enable_pre_call_checks = true
}
`
}
