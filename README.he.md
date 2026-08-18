# terraform-provider-homecloud

ספק Terraform / OpenTofu למשאבי חשבון HomeCloud. מדבר עם `console.{apex}/api/v1`
באמצעות **Access Keys ב-SigV1** (ADR-049). **לא** מנהל K3s, Helm או בתים ב-data plane.

הספק **עדיין לא** ב-Terraform Registry. בנו מקומית והשתמשו ב-`dev_overrides`
(ראו README באנגלית). רישום Registry: [`PUBLISHING.md`](PUBLISHING.md).

בקונסול: בכל משאב נתמך יש לשונית **Terraform** (או כפתור העתקה) שמעתיקה HCL
+ הערת `terraform import`. סודות לעולם לא מועתקים.

## משאבים

| משאב | הערה |
|------|------|
| `homecloud_mq_queue` | תור MQ |
| `homecloud_so_bucket` | באקט — `name` בלבד |
| `homecloud_secret` | `values` הם write-only; GET לא מחזיר ערכים |
| `homecloud_iam_policy` / `_iam_role` / `_iam_policy_attachment` | דורש owner/admin |
| `homecloud_mdb_instance` / `_mdb_user` / `_redis_instance` | Create ממתין ל-`status=active` |
| `homecloud_function` / `_function_url` | בלי קבצי workspace; מחיקת פונקציה = owner/admin |
| `homecloud_ir_repository` | בלי תגיות image |
| `homecloud_domain` | אימות TXT מחוץ ל-Terraform |
| `homecloud_compute_machine` | ממתין ל-Operations |
| `homecloud_ssh_key` | מפתח פרטי רק ביצירה |
| `homecloud_application` | spec/`draft` בלבד |

מקור הספק: `homecloudlab/homecloud`. הרשאות: `HC_ACCESS_KEY_ID` + `HC_SECRET_ACCESS_KEY`.
