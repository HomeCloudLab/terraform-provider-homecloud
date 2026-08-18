package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"homecloud": providerserver.NewProtocol6WithError(New("acc")()),
	}
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 with HC_ACCESS_KEY_ID / HC_SECRET_ACCESS_KEY / HC_APEX to run acceptance tests")
	}
	if os.Getenv("HC_ACCESS_KEY_ID") == "" || os.Getenv("HC_SECRET_ACCESS_KEY") == "" {
		t.Fatal("TF_ACC=1 requires HC_ACCESS_KEY_ID and HC_SECRET_ACCESS_KEY")
	}
}

func testAccMVPConfig(suffix string) string {
	return fmt.Sprintf(`
provider "homecloud" {}

data "homecloud_account" "this" {}

resource "homecloud_mq_queue" "demo" {
  name = "tfacc-%[1]s-q"
}

resource "homecloud_so_bucket" "demo" {
  name = "tfacc-%[1]s-b"
}
`, suffix)
}

func TestAccMVPQueueBucketAccount(t *testing.T) {
	testAccPreCheck(t)
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	queueName := "tfacc-" + suffix + "-q"
	bucketName := "tfacc-" + suffix + "-b"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccMVPConfig(suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.homecloud_account.this", "id"),
					resource.TestCheckResourceAttrSet("data.homecloud_account.this", "account_number"),
					resource.TestCheckResourceAttr("homecloud_mq_queue.demo", "name", queueName),
					resource.TestMatchResourceAttr(
						"homecloud_mq_queue.demo",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:mq::[0-9]+:queue/`+regexp.QuoteMeta(queueName)+`$`),
					),
					resource.TestCheckResourceAttr("homecloud_so_bucket.demo", "name", bucketName),
					resource.TestMatchResourceAttr(
						"homecloud_so_bucket.demo",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:so::[0-9]+:bucket/`+regexp.QuoteMeta(bucketName)+`$`),
					),
				),
			},
			{
				Config:             testAccMVPConfig(suffix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccSecretConfig(suffix string) string {
	return fmt.Sprintf(`
provider "homecloud" {}

resource "homecloud_secret" "demo" {
  name        = "tfacc-%[1]s-secret"
  description = "acc"
  values = {
    EXAMPLE_KEY = "acc-value"
  }
}
`, suffix)
}

func TestAccSecret(t *testing.T) {
	testAccPreCheck(t)
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	name := "tfacc-" + suffix + "-secret"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccSecretConfig(suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homecloud_secret.demo", "name", name),
					resource.TestMatchResourceAttr(
						"homecloud_secret.demo",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:secrets::[0-9]+:secret/`+regexp.QuoteMeta(name)+`$`),
					),
					resource.TestCheckResourceAttr("homecloud_secret.demo", "has_value", "true"),
				),
			},
			{
				Config:             testAccSecretConfig(suffix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
