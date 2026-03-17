use std::sync::Mutex;
use std::time::Duration;

use tauri::Manager;
use tauri::RunEvent;
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_shell::process::CommandChild;
use tauri_plugin_shell::ShellExt;

const BACKEND_PORT: u16 = 3001;
const HEALTH_URL: &str = "http://localhost:3001/api/sessions";
const HEALTH_POLL_INTERVAL: Duration = Duration::from_millis(200);
const HEALTH_TIMEOUT: Duration = Duration::from_secs(10);

struct SidecarState(Mutex<Option<CommandChild>>);

fn wait_for_backend() -> Result<(), String> {
    let start = std::time::Instant::now();
    let client = reqwest::blocking::Client::builder()
        .timeout(Duration::from_secs(1))
        .build()
        .map_err(|e| format!("failed to create HTTP client: {e}"))?;

    loop {
        if start.elapsed() > HEALTH_TIMEOUT {
            return Err(format!(
                "backend did not become ready within {}s",
                HEALTH_TIMEOUT.as_secs()
            ));
        }

        match client.get(HEALTH_URL).send() {
            Ok(resp) if resp.status().is_success() => return Ok(()),
            _ => std::thread::sleep(HEALTH_POLL_INTERVAL),
        }
    }
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            None,
        ))
        .setup(|app| {
            // Spawn the Go backend as a sidecar process
            let sidecar = app
                .shell()
                .sidecar("cmux-server")
                .map_err(|e| format!("failed to create sidecar command: {e}"))?;

            let (_rx, child) = sidecar
                .spawn()
                .map_err(|e| format!("failed to spawn sidecar: {e}"))?;

            app.manage(SidecarState(Mutex::new(Some(child))));

            // Wait for the backend to be ready before showing the window
            wait_for_backend().map_err(|e| {
                eprintln!("Backend health check failed: {e}");
                e
            })?;

            // Navigate the main window to the backend URL
            if let Some(window) = app.get_webview_window("main") {
                let url = format!("http://localhost:{BACKEND_PORT}");
                let _ = window.navigate(url.parse().unwrap());
            }

            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error building tauri application")
        .run(|app, event| {
            if let RunEvent::Exit = event {
                // Kill the sidecar when the app exits
                if let Some(state) = app.try_state::<SidecarState>() {
                    if let Ok(mut guard) = state.0.lock() {
                        if let Some(child) = guard.take() {
                            let _ = child.kill();
                        }
                    }
                }
            }
        });
}
