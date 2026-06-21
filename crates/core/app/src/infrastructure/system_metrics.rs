use sysinfo::{Disks, Networks, Pid, ProcessesToUpdate, System};

use crate::domain::system::ServerResourceMetrics;
use crate::service::system::SystemMetricsCollector;

#[derive(Default)]
pub struct SysinfoMetricsCollector;

impl SystemMetricsCollector for SysinfoMetricsCollector {
    fn collect(&self) -> ServerResourceMetrics {
        let mut system = System::new_all();
        system.refresh_memory();
        system.refresh_cpu_all();
        let current_pid = Pid::from_u32(std::process::id());
        let current_processes = [current_pid];
        system.refresh_processes(ProcessesToUpdate::Some(&current_processes), true);
        let current_process = system.process(current_pid);
        let process_cpu_usage_percent = current_process.map_or(0.0, |process| process.cpu_usage());
        let process_memory_bytes = current_process.map_or(0, |process| process.memory());
        let process_virtual_memory_bytes =
            current_process.map_or(0, |process| process.virtual_memory());

        let disks = Disks::new_with_refreshed_list();
        let disk_count = disks.iter().count() as u64;
        let (total_disk_bytes, available_disk_bytes) =
            disks
                .iter()
                .fold((0_u64, 0_u64), |(total, available), disk| {
                    (
                        total.saturating_add(disk.total_space()),
                        available.saturating_add(disk.available_space()),
                    )
                });
        let used_disk_bytes = total_disk_bytes.saturating_sub(available_disk_bytes);
        let networks = Networks::new_with_refreshed_list();
        let network_interface_count = networks.iter().count() as u64;
        let (network_received_bytes, network_transmitted_bytes) =
            networks
                .iter()
                .fold((0_u64, 0_u64), |(received, transmitted), (_, data)| {
                    (
                        received.saturating_add(data.total_received()),
                        transmitted.saturating_add(data.total_transmitted()),
                    )
                });
        let load_average = System::load_average();

        ServerResourceMetrics {
            source: "sysinfo".into(),
            cpu_usage_percent: system.global_cpu_usage(),
            process_cpu_usage_percent,
            total_memory_bytes: system.total_memory(),
            used_memory_bytes: system.used_memory(),
            available_memory_bytes: system.available_memory(),
            process_memory_bytes,
            process_virtual_memory_bytes,
            total_swap_bytes: system.total_swap(),
            used_swap_bytes: system.used_swap(),
            total_disk_bytes,
            used_disk_bytes,
            available_disk_bytes,
            disk_count,
            network_interface_count,
            network_received_bytes,
            network_transmitted_bytes,
            system_uptime_seconds: System::uptime(),
            system_boot_time_seconds: System::boot_time(),
            load_average_one: load_average.one,
            load_average_five: load_average.five,
            load_average_fifteen: load_average.fifteen,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn collector_returns_realistic_resource_bounds() {
        let metrics = SysinfoMetricsCollector.collect();

        assert_eq!(metrics.source, "sysinfo");
        assert!(metrics.cpu_usage_percent >= 0.0);
        assert!(metrics.process_cpu_usage_percent >= 0.0);
        assert!(metrics.total_memory_bytes >= metrics.used_memory_bytes);
        assert!(metrics.total_memory_bytes >= metrics.available_memory_bytes);
        assert!(metrics.process_memory_bytes <= metrics.total_memory_bytes);
        assert!(metrics.process_virtual_memory_bytes >= metrics.process_memory_bytes);
        assert!(metrics.total_swap_bytes >= metrics.used_swap_bytes);
        assert_eq!(
            metrics.used_disk_bytes,
            metrics
                .total_disk_bytes
                .saturating_sub(metrics.available_disk_bytes)
        );
        let _ = metrics.network_interface_count;
        let _ = metrics.network_received_bytes;
        let _ = metrics.network_transmitted_bytes;
        assert!(metrics.system_uptime_seconds > 0);
        assert!(metrics.system_boot_time_seconds > 0);
        assert!(metrics.load_average_one >= 0.0);
        assert!(metrics.load_average_five >= 0.0);
        assert!(metrics.load_average_fifteen >= 0.0);
    }
}
