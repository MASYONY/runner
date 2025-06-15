# Fabi-Runner Dokumentation

## Übersicht

Fabi-Runner ist ein modularer, YAML-basierter Job-Runner für verschiedene Workloads (Container, VMs, APIs, Skripte etc.). Er unterstützt verschiedene Executor-Typen (Docker, Local, SSH, Proxmox, Lexware, sevDesk, Custom) und ist einfach erweiterbar.

---

## Grundstruktur eines Jobs

```yaml
job_id: <eindeutige ID>
executor: <docker|local|ssh|proxmox|lexware|sevdesk|custom>
type: <job-typ/shortcut>
product: <executor-spezifische Felder>
artifacts: # optional
  - path: <Pfad/Wildcard>
    type: <file|dir>
variables: # optional
  KEY: VALUE
callback: # optional
  url: <Callback-URL>
  secret: <Callback-Secret>
```

---

## Executor-Typen & Shortcuts

### Docker
- image, before_script, script, commands, namespace, mounts, env, tty

### Local
- commands (String oder Array)

### SSH
- host, user, commands (String oder Array), port, keyfile

### Custom
- script (String oder Array)

### Proxmox
- host, node, token_id, token_secret, vmid, type, api_command, api_params
- Shortcuts: lxc_create, kvm_create, lxc_start, kvm_stop, lxc_delete, kvm_status, lxc_status, kvm_config, lxc_config, kvm_agent_exec, lxc_agent_exec, kvm_vncproxy, lxc_vncproxy, kvm_vncwebsocket, lxc_vncwebsocket, kvm_migrate, lxc_migrate, kvm_clone, lxc_clone, kvm_resize, lxc_resize, kvm_firewall, lxc_firewall, kvm_metrics, lxc_metrics, kvm_list, lxc_list, kvm_snapshot, lxc_snapshot

### Lexware
- create_invoice: api_key, customer_id, invoice_data
- cancel_invoice: api_key, invoice_id

### sevDesk
- create_invoice: api_token, contact_id, invoice_data
- cancel_invoice: api_token, invoice_id

---

## Globale und Job-Variablen

- `variables:`: Beliebige Key-Value-Paare, z.B. für Umgebungsvariablen, TTY, etc.
- `artifacts:`: Liste von Dateien/Verzeichnissen, die nach dem Job kopiert werden (Wildcards möglich)
- `callback:`: Optionale URL & Secret für Status-Callback (z.B. Webhook)

---

## Konfigurationsdatei (config.yaml)

```yaml
default_callback_url: "https://example.com/callback"
default_callback_secret: "<SECRET>"
global_before_script:
  - echo "Starte Job..."
workdir: "workdir/"
logdir: "logs/"
```

- Wird automatisch geladen, falls kein --config angegeben ist.
- Globale Werte können pro Job überschrieben werden.

---

## Beispiele

Siehe `tests/<executor>/` für viele Beispiel-Jobs zu allen Executor-Typen und Proxmox-Shortcuts.

---

## Hinweise
- YAML: Bei Listen von Strings (z.B. script, commands) ggf. Strings in einfache Anführungszeichen setzen.
- Proxmox: Für API-Shortcuts reicht meist ein Minimal-Job, z.B. nur host, node, token_id, token_secret, vmid.
- Logging: Logs werden mit Logger-Präfix ausgegeben, optional auch an einen Socket (RUNNER_LOG_SOCKET).
- Status: Statusdatei wird bei jedem Statuswechsel aktualisiert.
- Artefakte: Nur explizit definierte Artefakte werden kopiert.

---

## Logging & Log-Socket

- Alle Logs werden mit Logger-Präfix (INFO/ERROR) ausgegeben und im Logverzeichnis gespeichert.
- Zusätzlich kann die Umgebungsvariable `RUNNER_LOG_SOCKET` gesetzt werden:
  - Beispiel: `RUNNER_LOG_SOCKET=/tmp/runner.sock`
  - Dann werden alle Logs zusätzlich an diesen Unix Domain Socket gesendet (z.B. für zentrale Log-Aggregation oder Live-Viewer).
- Der Socket muss vor dem Start des Runners existieren und erreichbar sein.
- Die Log-Ausgabe bleibt weiterhin auch auf der Konsole und im Logfile erhalten.

---

## Erweiterbarkeit
- Neue Executor-Typen können einfach als Go-Datei im `executors/`-Ordner ergänzt werden.
- Shortcuts für Proxmox/Jira/andere APIs können in der Dispatch-Logik in `jobs/job.go` ergänzt werden.

---

## Unterstützte Umgebungsvariablen

| Variable              | Bedeutung / Wirkung                                              |
|-----------------------|-----------------------------------------------------------------|
| RUNNER_ID             | Eindeutige ID des Runners (optional, für Logging/Tracing)        |
| RUNNER_HOSTNAME       | Hostname des Runners (optional, für Logging/Tracing)             |
| RUNNER_WORKDIR        | Arbeitsverzeichnis für Jobs (Default: ./workdir)                 |
| RUNNER_LOG_DIR        | Verzeichnis für Logs (Default: ./logs)                           |
| RUNNER_LOG_SOCKET     | Pfad zu Unix Domain Socket für Log-Forwarding (optional)         |

Diese Variablen können beim Start des Runners gesetzt werden und beeinflussen Verhalten, Logging und Pfade.

---

## Fragen oder Erweiterungswünsche?
Melde dich direkt im Projekt oder erstelle ein Issue!
