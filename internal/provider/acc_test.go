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

func testAccIAMConfig(suffix string) string {
	return fmt.Sprintf(`
provider "homecloud" {}

data "homecloud_account" "this" {}

data "homecloud_iam_service_account" "functions" {
  name = "functions"
}

resource "homecloud_iam_policy" "mq" {
  name        = "tfacc-%[1]s-mq"
  description = "acc"
  document = jsonencode({
    Version = "2026-07-24"
    Statement = [{
      Effect   = "Allow"
      Action   = ["mq:*"]
      Resource = "arn:homecloud:mq::${data.homecloud_account.this.account_number}:queue/*"
    }]
  })
}

resource "homecloud_iam_role" "ci" {
  name = "tfacc-%[1]s-ci"
}

resource "homecloud_iam_policy_attachment" "functions_mq" {
  policy_arn     = homecloud_iam_policy.mq.arn
  principal_type = "service_account"
  principal_id   = data.homecloud_iam_service_account.functions.id
}
`, suffix)
}

func TestAccIAM(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("HC_TF_ACC_IAM") == "" {
		t.Skip("set HC_TF_ACC_IAM=1 with an owner/admin Access Key (iam.manage)")
	}
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	policyName := "tfacc-" + suffix + "-mq"
	roleName := "tfacc-" + suffix + "-ci"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccIAMConfig(suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homecloud_iam_policy.mq", "name", policyName),
					resource.TestMatchResourceAttr(
						"homecloud_iam_policy.mq",
						"arn",
						regexp.MustCompile(`^arn:homecloud:iam::[0-9]+:policy/`+regexp.QuoteMeta(policyName)+`$`),
					),
					resource.TestCheckResourceAttr("homecloud_iam_role.ci", "name", roleName),
					resource.TestCheckResourceAttrSet("homecloud_iam_policy_attachment.functions_mq", "id"),
					resource.TestCheckResourceAttrSet("data.homecloud_iam_service_account.functions", "id"),
				),
			},
			{
				Config:             testAccIAMConfig(suffix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccMDBConfig(suffix string) string {
	return fmt.Sprintf(`
provider "homecloud" {}

resource "homecloud_mdb_instance" "app" {
  name           = "tfacc-%[1]s-pg"
  engine         = "postgresql"
  instance_class = "micro"
}

resource "homecloud_mdb_user" "ci" {
  instance_name = homecloud_mdb_instance.app.name
  username      = "ci"
  password      = "ChangeMe22"
  role          = "readwrite"
}

resource "homecloud_redis_instance" "cache" {
  name           = "tfacc-%[1]s-redis"
  instance_class = "micro"
}
`, suffix)
}

func TestAccMDBRedis(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("HC_TF_ACC_MDB") == "" {
		t.Skip("set HC_TF_ACC_MDB=1 to run MDB/Redis acceptance tests (waiters can take several minutes)")
	}
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	dbName := "tfacc-" + suffix + "-pg"
	cacheName := "tfacc-" + suffix + "-redis"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccMDBConfig(suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homecloud_mdb_instance.app", "name", dbName),
					resource.TestCheckResourceAttr("homecloud_mdb_instance.app", "status", "active"),
					resource.TestMatchResourceAttr(
						"homecloud_mdb_instance.app",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:mdb::[0-9]+:instance/`+regexp.QuoteMeta(dbName)+`$`),
					),
					resource.TestCheckResourceAttr("homecloud_mdb_user.ci", "username", "ci"),
					resource.TestCheckResourceAttr("homecloud_redis_instance.cache", "name", cacheName),
					resource.TestCheckResourceAttr("homecloud_redis_instance.cache", "status", "active"),
				),
			},
			{
				Config:             testAccMDBConfig(suffix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
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

func testAccP4Config(suffix string) string {
	return fmt.Sprintf(`
provider "homecloud" {}

resource "homecloud_function" "demo" {
  name    = "tfacc-%[1]s-fn"
  handler = "main.handler"
}

resource "homecloud_function_url" "demo" {
  function_name = homecloud_function.demo.name
}

resource "homecloud_ir_repository" "demo" {
  name = "tfacc-%[1]s"
}

resource "homecloud_domain" "demo" {
  hostname = "tfacc-%[1]s.example.com"
}
`, suffix)
}

func TestAccP4FunctionRegistryDomain(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("HC_TF_ACC_P4") == "" {
		t.Skip("set HC_TF_ACC_P4=1 to run function/IR/domain acceptance tests")
	}
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	fnName := "tfacc-" + suffix + "-fn"
	repoName := "tfacc-" + suffix
	hostname := "tfacc-" + suffix + ".example.com"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccP4Config(suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homecloud_function.demo", "name", fnName),
					resource.TestMatchResourceAttr(
						"homecloud_function.demo",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:functions::[0-9]+:function/`+regexp.QuoteMeta(fnName)+`$`),
					),
					resource.TestCheckResourceAttr("homecloud_function_url.demo", "function_name", fnName),
					resource.TestCheckResourceAttrSet("homecloud_function_url.demo", "function_url"),
					resource.TestCheckResourceAttr("homecloud_ir_repository.demo", "name", repoName),
					resource.TestMatchResourceAttr(
						"homecloud_ir_repository.demo",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:ir::[0-9]+:repository/`+regexp.QuoteMeta(repoName)+`$`),
					),
					resource.TestCheckResourceAttr("homecloud_domain.demo", "hostname", hostname),
					resource.TestCheckResourceAttr("homecloud_domain.demo", "status", "pending_verification"),
					resource.TestMatchResourceAttr(
						"homecloud_domain.demo",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:domains::[0-9]+:domain/`+regexp.QuoteMeta(hostname)+`$`),
					),
				),
			},
			{
				Config:             testAccP4Config(suffix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccP5Config(suffix string) string {
	return fmt.Sprintf(`
provider "homecloud" {}

resource "homecloud_ssh_key" "demo" {
  name = "tfacc-%[1]s"
}

resource "homecloud_application" "demo" {
  name     = "tfacc-%[1]s"
  slug     = "tfacc-%[1]s"
  template = "api-only"
}
`, suffix)
}

func TestAccP5SSHKeyAndApplication(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("HC_TF_ACC_P5") == "" {
		t.Skip("set HC_TF_ACC_P5=1 to run SSH key and application acceptance tests")
	}
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	name := "tfacc-" + suffix

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccP5Config(suffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homecloud_ssh_key.demo", "name", name),
					resource.TestMatchResourceAttr(
						"homecloud_ssh_key.demo",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:compute::[0-9]+:ssh-key/`+regexp.QuoteMeta(name)+`$`),
					),
					resource.TestCheckResourceAttrSet("homecloud_ssh_key.demo", "private_key"),
					resource.TestCheckResourceAttr("homecloud_application.demo", "slug", name),
					resource.TestCheckResourceAttr("homecloud_application.demo", "status", "draft"),
					resource.TestMatchResourceAttr(
						"homecloud_application.demo",
						"iam_arn",
						regexp.MustCompile(`^arn:homecloud:applications::[0-9]+:application/`+regexp.QuoteMeta(name)+`$`),
					),
				),
			},
			{
				Config:             testAccP5Config(suffix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccP5ComputeMachine(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("HC_TF_ACC_COMPUTE") == "" {
		t.Skip("set HC_TF_ACC_COMPUTE=1 to run compute machine acceptance tests (provisions a VM)")
	}
	suffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	name := "tfacc-" + suffix
	config := fmt.Sprintf(`
provider "homecloud" {}

resource "homecloud_compute_machine" "demo" {
  name          = "%[1]s"
  machine_class = "basic"
  image_id      = "ubuntu-24.04"
}
`, name)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homecloud_compute_machine.demo", "name", name),
					resource.TestCheckResourceAttr("homecloud_compute_machine.demo", "status", "RUNNING"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
