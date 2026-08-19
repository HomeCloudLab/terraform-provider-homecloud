# Terraform / OpenTofu ל-HomeCloud

מדריך משתמש לספק `homecloud` (`source = "homecloudlab/homecloud"`).
English: [docs/guides/getting-started.md](docs/guides/getting-started.md).

הספק מנהל **משאבי חשבון** ב-`console.{apex}/api/v1` עם **Access Keys ב-SigV1**.
הוא **לא** מנהל את ה-homelab עצמו (K3s, Helm, Traefik, GitOps).

רשום ב-Terraform Registry:
[`homecloudlab/homecloud`](https://registry.terraform.io/providers/homecloudlab/homecloud/latest)
(**v0.1.0**). הריצו `terraform init`. CI ו-`release.yml` רצים ב-GitHub Actions —
ראו [`PUBLISHING.md`](PUBLISHING.md).

ריפו: [`terraform-provider-homecloud`](https://github.com/HomeCloudLab/terraform-provider-homecloud)
(GitHub עלול להציג רמז לשינוי שם; השאירו את הקידומת `terraform-provider-*` בשביל ה-Registry).

---

## אימות

צרו **Access Key הקשור למשתמש** בקונסול (IAM → Access keys). את הסוד שימו
במשתני סביבה — לא בקבצי `.tf`.

| משתנה | משמעות |
|--------|--------|
| `HC_ACCESS_KEY_ID` | מזהה המפתח |
| `HC_SECRET_ACCESS_KEY` | הסוד |
| `HC_APEX` | Apex של הפלטפורמה (ברירת מחדל `holab.abrdns.com`) |
| `HC_ACCOUNT_ID` | UUID חשבון אופציונלי (ברירת מחדל: whoami) |
| `HC_ENDPOINT` | כתובת console חלופית (בדיקות) |

יצירה/עדכון/מחיקה של IAM דורשים תפקיד קונסול **owner או admin**. גם **מחיקת פונקציה**
דורשת owner/admin. יצירה/מחיקה/קריאה של תור/באקט/סוד יכולות עם מפתח Service Account
ופעולות IAM מתאימות. נתיבי קונסול שלא מופו ל-SA מחזירים `403 iam.management_sa_not_enabled`.

### GitHub OIDC (בלי מפתח ארוך-טווח)

ב-CI אפשר להנפיק Access Key **זמני** מ-JWT של GitHub Actions. אין צורך ב-
`HC_SECRET_ACCESS_KEY` בסודות הריפו.

1. צרו IAM role שה-**trust** שלו מתיר GitHub Actions (`Principal.Federated` +
   `Condition` על `sub` ו-`aud`). צרפו מדיניות מנוהלת כמו `MQAdmin` /
   `SOBucketAdmin` / `SecretsAdmin`.
2. ב-GitHub Actions: `permissions: id-token: write`.
3. הגדירו `HC_ROLE_ARN`. הספק מחליף את ה-JWT ב-
   `POST /api/v1/sts/assume-role-with-web-identity` ומשתמש באישורי SigV1
   קצרי-חיים (כולל `X-Homecloud-Session-Token`).

סשן assumed-role זהה לנתיבי Service Account הממופים (תור/באקט/סוד). נתיבים
שלא מופו מחזירים `403 iam.management_role_not_enabled`. IAM / MDB / פונקציות /
מחשוב עדיין דורשים מפתח הקשור למשתמש.

```json
{
  "Version": "2026-07-24",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:homecloud:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
    },
    "Action": "sts:AssumeRole",
    "Condition": {
      "StringEquals": {
        "token.actions.githubusercontent.com:aud": "https://console.holab.abrdns.com"
      },
      "StringLike": {
        "token.actions.githubusercontent.com:sub": "repo:HomeCloudLab/my-infra:*"
      }
    }
  }]
}
```

ראו [examples/github-oidc](examples/github-oidc).

| משתנה | משמעות |
|--------|--------|
| `HC_ROLE_ARN` | ARN של ה-Role להנחה |
| `HC_WEB_IDENTITY_TOKEN` | JWT של OIDC (אופציונלי ב-Actions) |
| `HC_OIDC_AUDIENCE` | ה-audience של ה-JWT (ברירת מחדל `https://console.{apex}`) |
| `HC_SESSION_TOKEN` | טוקן STS שכבר הוחלף |

---

## התקנה

`terraform init` מוריד **v0.1.0** מה-Registry. אין צורך בבינארי מקומי.

לעבוד על הריפו הזה: בנו מקומית והשתמשו ב-`dev_overrides` (ואז דלגו על `init`):

```powershell
cd terraform-provider-homecloud
go build -o terraform-provider-homecloud.exe .
Copy-Item dev.tfrc.example dev.tfrc
# ערכו את dev.tfrc: נתיב מלא לתיקייה (סלאשים קדימה בסדר).
$env:TF_CLI_CONFIG_FILE = "$PWD\dev.tfrc"
$env:HC_ACCESS_KEY_ID = "HCAK..."
$env:HC_SECRET_ACCESS_KEY = "..."
$env:HC_APEX = "holab.abrdns.com"
```

`dev_overrides` לא משתמשים ב-Registry. **דלגו על `terraform init`** כל עוד הם
מוגדרים. `terraform apply` מזהיר ש-overrides פעילים. זה צפוי.

```hcl
terraform {
  required_providers {
    homecloud = {
      source = "homecloudlab/homecloud"
    }
  }
}

provider "homecloud" {
  apex = "holab.abrdns.com"
}
```

OpenTofu עובד אותו דבר (`tofu apply`).

---

## קטלוג משאבים

כמעט הכול מיובא לפי **שם** (גם UUID במקומות שה-API מקבל). אחרי import,
`terraform plan` אמור להיות ריק אם ה-HCL תואם לאובייקט החי.

### `data.homecloud_account`

החשבון הנוכחי ממפתח הגישה (או `id` מפורש).

```hcl
data "homecloud_account" "this" {}

output "account_number" {
  value = data.homecloud_account.this.account_number
}
```

לקריאה בלבד: `id`, `short_id`, `account_number`, `name`, `slug`, `status`.

### `homecloud_mq_queue`

תור MQ. אופציונלי: `max_message_size`, `visibility_timeout_seconds`,
`max_receive_count`, `message_retention_seconds`. מחושב: `iam_arn`, `queue_url`,
`status`.

```hcl
resource "homecloud_mq_queue" "jobs" {
  name = "jobs"
}

# terraform import homecloud_mq_queue.jobs jobs
```

### `homecloud_so_bucket`

באקט ב-Object Storage. הסכימה היא **`name` בלבד**. Versioning, lifecycle, אתר
ומדיניות באקט נשארים בקונסול — לא שדות על המשאב הזה.

```hcl
resource "homecloud_so_bucket" "assets" {
  name = "assets"
}

# terraform import homecloud_so_bucket.assets assets
```

### `homecloud_secret`

מטא-דאטה של סוד. `values` הם **write-only** (Terraform 1.11+). GET אף פעם לא
מחזיר ערכים; הם לא נשמרים ב-state.

```hcl
resource "homecloud_secret" "db" {
  name = "db-creds"
  values = {
    username = "app"
    password = var.db_password
  }
}

# terraform import homecloud_secret.db db-creds
```

Import לא מחזיר את `values`. אם Terraform אמור לנהל את המטען, הגדירו אותם שוב ב-apply הבא.

### IAM

דורש owner/admin. ב-JSON של מדיניות, `Version` הוא **`2026-07-24`** (לא AWS
`2012-10-17`). `arn` של מדיניות כבר בצורת IAM (אין `iam_arn` נפרד).

```hcl
data "homecloud_iam_service_account" "functions" {
  name = "functions"
}

resource "homecloud_iam_policy" "mq" {
  name        = "mq-send"
  description = "שליחה לכל התורים"
  document = jsonencode({
    Version = "2026-07-24"
    Statement = [{
      Effect   = "Allow"
      Action   = ["mq:SendMessage"]
      Resource = "arn:homecloud:mq::${data.homecloud_account.this.account_number}:queue/*"
    }]
  })
}

resource "homecloud_iam_role" "ci" {
  name = "ci"
}

resource "homecloud_iam_policy_attachment" "functions_mq" {
  policy_arn     = homecloud_iam_policy.mq.arn
  principal_type = "service_account" # user | role | service_account
  principal_id   = data.homecloud_iam_service_account.functions.id
}

# terraform import homecloud_iam_policy.mq mq-send
# terraform import homecloud_iam_role.ci ci
# terraform import homecloud_iam_policy_attachment.functions_mq service_account:<sa-uuid>:<policy-arn>
```

`trust_document` בתפקיד הוא JSON אופציונלי (ברירת מחדל: Service Account בשם
`functions`). צירופים משתמשים ב-**UUID של ה-principal**, לא בשם התצוגה.

דוגמה: `examples/iam`.

### מסד מנוהל (`homecloud_mdb_instance` / `homecloud_mdb_user`)

יצירה **ממתינה** עד `status=active` (נכשלת ב-`failed`). מנוע:
`postgresql`, `mysql`, או `mongodb`. אופציונלי: `instance_class`, `engine_version`,
`storage_gi`, `database`, `owner`.

`password` של משתמש הוא write-only. ייבוא משתמש: `instance_name/username`.

```hcl
resource "homecloud_mdb_instance" "app" {
  name           = "app-pg"
  engine         = "postgresql"
  instance_class = "micro"
}

resource "homecloud_mdb_user" "ci" {
  instance_name = homecloud_mdb_instance.app.name
  username      = "ci"
  password      = var.db_password
  role          = "readwrite"
}

# terraform import homecloud_mdb_instance.app app-pg
# terraform import homecloud_mdb_user.ci app-pg/ci
```

דוגמה: `examples/mdb`.

### Redis (`homecloud_redis_instance`)

יצירה ממתינה ל-`status=active`. הסיסמה נמצאת ב-`credentials_secret`
(סוד HomeCloud), **לא** במשאב הזה.

```hcl
resource "homecloud_redis_instance" "cache" {
  name           = "app-redis"
  instance_class = "micro"
}

# terraform import homecloud_redis_instance.cache app-redis
```

### פונקציות (`homecloud_function` / `homecloud_function_url`)

**Spec בלבד:** runtime/handler/memory/timeout. Terraform **לא** מנהל קבצי
workspace, deploys, layers או invoke. מחיקה דורשת owner/admin (`function.delete`).

```hcl
resource "homecloud_function" "hello" {
  name    = "hello"
  handler = "main.handler"
}

resource "homecloud_function_url" "hello" {
  function_name      = homecloud_function.hello.name
  public_url_enabled = false
}

# terraform import homecloud_function.hello hello
# terraform import homecloud_function_url.hello hello
```

דוגמה: `examples/p4`.

### Image Registry (`homecloud_ir_repository`)

רשומת repository בלבד. **תגיות image אינן משאבי Terraform.**

```hcl
resource "homecloud_ir_repository" "app" {
  name = "app"
}

# terraform import homecloud_ir_repository.app app
```

### דומיין (`homecloud_domain`)

יצירה נשארת `pending_verification` עד שתוסיפו את רשומת ה-TXT **מחוץ ל-Terraform**.
אין המתנה ל-verify.

```hcl
resource "homecloud_domain" "site" {
  hostname = "app.example.com"
  dns_mode = "external" # או homecloud
}

# terraform import homecloud_domain.site app.example.com
```

### Compute (`homecloud_compute_machine` / `homecloud_ssh_key`)

יצירת מכונה ממתינה ל-Operations API (`SUCCEEDED` / `FAILED`).
**לא** מנהל firewall, דיסקים, exec או קבצים.

`homecloud_ssh_key.private_key` רגיש, מוחזר **פעם אחת ביצירה**, אף פעם לא ב-GET.
שמרו אותו מפלט ה-apply אם צריך.

```hcl
resource "homecloud_ssh_key" "ci" {
  name = "ci"
}

resource "homecloud_compute_machine" "web" {
  name          = "web-1"
  machine_class = "basic" # או standard
  image_id      = "ubuntu-24.04"
  ssh_key_ids   = [homecloud_ssh_key.ci.id]
}

# terraform import homecloud_ssh_key.ci ci
# terraform import homecloud_compute_machine.web web-1
```

`machine_class` נשלח ל-API כ-JSON `class`. Images: `ubuntu-24.04`,
`debian-12`, `almalinux-9`.

דוגמה: `examples/p5` (משאב המכונה בהערה — הוא מקים VM).

### אפליקציה (`homecloud_application`)

**Spec / `draft` בלבד.** בלי provision, deploy, scale או YAML apply.

```hcl
resource "homecloud_application" "api" {
  name     = "Shop"
  slug     = "shop"
  template = "api-only" # fullstack | static-site | worker
}

# terraform import homecloud_application.api shop
```

ייבוא לפי **slug**.

---

## מה נשאר בקונסול

אלה **לא** משאבי Terraform (משאבי אח בסגנון AWS, בהמשך):

- Versioning / lifecycle / אתר / מדיניות באקט
- קוד פונקציה, גרסאות, deploys, layers, invoke
- תגיות IR וכללי lifecycle
- רשומות DNS ואימות דומיין
- Firewall, דיסקים, snapshots ו-exec של מכונה
- Provision / deploy / HPA / דומיין מותאם של אפליקציה

הקונסול נשאר פתוח לכתיבה. Drift מטופל ב-`terraform plan`, `import`, או
`lifecycle.ignore_changes`. אין נעילה מסוג `managed_by=terraform`.

---

## דוגמאות בריפו של הספק

| תיקייה | מה נוצר |
|--------|---------|
| `examples/mvp` | תור + באקט + data של חשבון |
| `examples/secret` | סוד עם values ב-write-only |
| `examples/iam` | מדיניות + תפקיד + צירוף ל-SA (owner/admin) |
| `examples/mdb` | PostgreSQL + משתמש + Redis |
| `examples/p4` | פונקציה + URL + ריפו IR + דומיין |
| `examples/p5` | מפתח SSH + אפליקציה draft (מכונה בהערה) |
| `examples/github-oidc` | trust של GitHub Actions OIDC + שלד workflow |

---

## רישום ב-Registry

חי: [`registry.terraform.io/providers/homecloudlab/homecloud`](https://registry.terraform.io/providers/homecloudlab/homecloud/latest).
תגיות `v*` חדשות נחתמות ב-GPG. ראו [`PUBLISHING.md`](PUBLISHING.md).
