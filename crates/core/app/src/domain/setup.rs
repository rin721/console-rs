use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupStatus {
    pub completed: bool,
    pub has_initial_admin: bool,
    pub required_steps: Vec<SetupStepStatus>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupStepStatus {
    pub key: String,
    pub title: String,
    pub status: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupSchema {
    pub locale: String,
    pub steps: Vec<SetupStepSchema>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupConfigCheckSummary {
    pub ready: bool,
    pub checks: Vec<SetupConfigCheck>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupConfigCheck {
    pub key: String,
    pub title: String,
    pub status: String,
    pub severity: String,
    pub message: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupStepSchema {
    pub key: String,
    pub title: String,
    pub fields: Vec<SetupFieldSchema>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupFieldSchema {
    pub key: String,
    pub label: String,
    pub kind: String,
    pub required: bool,
    pub sensitive: bool,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CreateSetupRunRequest {
    pub reason: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupRun {
    pub id: String,
    pub status: String,
    pub reason: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SetupStepLog {
    pub step_key: String,
    pub status: String,
    pub message: String,
    pub created_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CompleteSetupRequest {
    pub confirm: bool,
    pub run_id: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CompleteSetupResult {
    pub completed: bool,
}
