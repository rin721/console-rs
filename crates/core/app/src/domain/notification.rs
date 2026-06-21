use serde::Serialize;

#[derive(Clone, Debug, Serialize)]
pub struct NotificationOutboxItem {
    pub id: i64,
    pub org_id: Option<i64>,
    pub user_id: Option<i64>,
    pub product_code: String,
    pub channel: String,
    pub template_code: String,
    pub recipient: String,
    pub related_kind: String,
    pub related_id: i64,
    pub payload_json: String,
    pub status: String,
    pub available_at: String,
    pub created_at: String,
    pub locked_at: Option<String>,
    pub delivered_at: Option<String>,
    pub failed_at: Option<String>,
    pub failure_reason: Option<String>,
    pub attempt_count: i64,
    #[serde(skip_serializing)]
    pub delivery_secret_ciphertext: Option<String>,
}

#[derive(Clone, Debug, Serialize)]
pub struct NotificationDrainReport {
    pub driver: String,
    pub claimed: usize,
    pub delivered: usize,
    pub retried: usize,
    pub failed: usize,
}

#[derive(Clone, Debug)]
pub struct NotificationDeadLetterRecord {
    pub id: i64,
    pub org_id: Option<i64>,
    pub user_id: Option<i64>,
    pub product_code: String,
    pub channel: String,
    pub template_code: String,
    pub recipient: String,
    pub related_kind: String,
    pub related_id: i64,
    pub status: String,
    pub created_at: String,
    pub failed_at: Option<String>,
    pub failure_reason: Option<String>,
    pub attempt_count: i64,
    pub delivery_secret_status: Option<String>,
    pub delivery_secret_ciphertext: Option<String>,
    pub delivery_secret_purged_at: Option<String>,
}

#[derive(Clone, Debug, Serialize)]
pub struct NotificationDeadLetterEntry {
    pub id: i64,
    pub org_id: Option<i64>,
    pub user_id: Option<i64>,
    pub product_code: String,
    pub channel: String,
    pub template_code: String,
    pub recipient_hint: String,
    pub related_kind: String,
    pub related_id: i64,
    pub status: String,
    pub created_at: String,
    pub failed_at: Option<String>,
    pub failure_reason: Option<String>,
    pub attempt_count: i64,
    pub secret_state: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct NotificationDeadLetterReport {
    pub driver: String,
    pub total: usize,
    pub entries: Vec<NotificationDeadLetterEntry>,
}

#[derive(Clone, Debug, Serialize)]
pub struct NotificationRequeueEntry {
    pub id: i64,
    pub org_id: Option<i64>,
    pub user_id: Option<i64>,
    pub product_code: String,
    pub channel: String,
    pub template_code: String,
    pub recipient_hint: String,
    pub related_kind: String,
    pub related_id: i64,
    pub action: String,
    pub reason: String,
    pub secret_state: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct NotificationRequeueReport {
    pub driver: String,
    pub total: usize,
    pub requeued: usize,
    pub skipped: usize,
    pub entries: Vec<NotificationRequeueEntry>,
}

#[derive(Clone, Debug)]
pub struct NotificationFailureDisposition {
    pub retried: bool,
    pub failed: bool,
}

#[derive(Clone, Debug)]
pub struct NotificationDeliveryMessage {
    pub item: NotificationOutboxItem,
    pub secret_token: Option<String>,
}
