use std::time::{Duration, Instant};

use reqwest::header::USER_AGENT;
use serde_json::json;

use crate::domain::system::{TrafficProbeObservation, TrafficProbeTargetEntry};
use crate::service::system::TrafficProbeRunner;

#[derive(Clone)]
pub struct HttpTrafficProbeRunner {
    client: reqwest::Client,
}

impl HttpTrafficProbeRunner {
    pub fn new() -> Self {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(5))
            .build()
            .expect("HTTP traffic probe client should be constructible");
        Self { client }
    }
}

impl Default for HttpTrafficProbeRunner {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait::async_trait]
impl TrafficProbeRunner for HttpTrafficProbeRunner {
    async fn probe(&self, target: &TrafficProbeTargetEntry) -> TrafficProbeObservation {
        let started = Instant::now();
        let result = self
            .client
            .get(&target.url)
            .header(USER_AGENT, "Aoi traffic probe")
            .send()
            .await;
        let duration_ms = started.elapsed().as_millis().min(i64::MAX as u128) as i64;

        match result {
            Ok(response) => {
                let status_code = i64::from(response.status().as_u16());
                let matched = status_code == target.expected_status;
                let status = if matched { "healthy" } else { "warning" };
                TrafficProbeObservation {
                    status: status.into(),
                    detail: json!({
                        "source": "reqwest-http",
                        "method": "GET",
                        "target_id": target.id,
                        "expected_status": target.expected_status,
                        "status_code": status_code,
                        "duration_ms": duration_ms,
                        "final_url": response.url().as_str(),
                        "reason": if matched { "expected_status_matched" } else { "status_mismatch" },
                    }),
                }
            }
            Err(error) => {
                let reason = if error.is_timeout() {
                    "timeout"
                } else if error.is_connect() {
                    "connect_failed"
                } else {
                    "request_failed"
                };
                TrafficProbeObservation {
                    status: "critical".into(),
                    detail: json!({
                        "source": "reqwest-http",
                        "method": "GET",
                        "target_id": target.id,
                        "expected_status": target.expected_status,
                        "duration_ms": duration_ms,
                        "reason": reason,
                        "is_timeout": error.is_timeout(),
                        "is_connect": error.is_connect(),
                    }),
                }
            }
        }
    }
}
