# Modular Job Runner

Ein flexibler, modularer Job Runner zur Ausführung beliebiger Prozesse wie Hosting-Provisionierung, Rechnungsprozesse, Deployment oder individuelle Workflows. Der Runner ist als eigenständiger Service konzipiert, der Jobs über konfigurierbare YAML-Dateien entgegennimmt, ausführt und die Ergebnisse inklusive Logs und Artefakten verwaltet.

## Vision

Der Runner soll als **unabhängiger, modularer Service** arbeiten, der über eine API angesteuert werden kann. Dadurch ist er flexibel in verschiedenste Backend-Systeme integrierbar und ermöglicht eine individuelle Steuerung und Erweiterung von Abläufen – von Provisionierung bis Abrechnung – ohne monolithisches System.

Das folgende Diagramm zeigt die grundlegende Architektur des Systems mit den Komponenten Backend, Runner-Daemon und Runner.

[![](https://mermaid.ink/img/pako:eNqFUsFOg0AQ_ZXNHIwm2ACFghxMqniosYla9aB42MIWSMsuWXZja9N_dxZKW714IfPmzbx5w-wWUpExiCCXtC7IS5xwQho97-ANTZeMZyZHyPhx8pEAfsn5vZiTJ800s8gbk190pTTPLfI6uUjg01S3Tb-knjXnTMaUVYJ3erO0YJleMYmqh3jfT0hcNjVVmDb0Efyj3_XerVmqlTCdxmkHS5zbiz-IPG-FMWjIGRlLVS5oqhoypZzmrGJc_Z1kFr-8vD7aNskDaKkTm_x0g5bsTRmqj1uiMwMWVExWtMzwGFtTlIAq0EkCEYYZlcsEEr7DOoqtsw1PIVISTwBS6LyAaEFXDSJdZ1SxuKT4X6q-pKb8XYgDZFmJ06fd5dsHYOEDMKP3irg1k7dCcwWR4_itAERbWCMM_IEfDkPP9l3HG_keshuIwmAQBLY9RMYPPG8U7iz4bkfag9DzR64bXgWOFzpD1939AEbWy6Q?type=png)](https://mermaid.live/edit#pako:eNqFUsFOg0AQ_ZXNHIwm2ACFghxMqniosYla9aB42MIWSMsuWXZja9N_dxZKW714IfPmzbx5w-wWUpExiCCXtC7IS5xwQho97-ANTZeMZyZHyPhx8pEAfsn5vZiTJ800s8gbk190pTTPLfI6uUjg01S3Tb-knjXnTMaUVYJ3erO0YJleMYmqh3jfT0hcNjVVmDb0Efyj3_XerVmqlTCdxmkHS5zbiz-IPG-FMWjIGRlLVS5oqhoypZzmrGJc_Z1kFr-8vD7aNskDaKkTm_x0g5bsTRmqj1uiMwMWVExWtMzwGFtTlIAq0EkCEYYZlcsEEr7DOoqtsw1PIVISTwBS6LyAaEFXDSJdZ1SxuKT4X6q-pKb8XYgDZFmJ06fd5dsHYOEDMKP3irg1k7dCcwWR4_itAERbWCMM_IEfDkPP9l3HG_keshuIwmAQBLY9RMYPPG8U7iz4bkfag9DzR64bXgWOFzpD1939AEbWy6Q)

Beschreibung der Komponenten
Backend:
Erstellt und verwaltet Jobs über API oder Datenbank (z.B. via REST oder GraphQL).
Zeigt den Status, verwaltet Nutzer & Rechte.

Runner-Daemon:
Holt Jobs aus der Queue (z.B. Datenbank oder Message Broker).
Plant und verteilt Jobs an verfügbare Runner-Instanzen.
Überwacht Fortschritt und Status.

Runner:
Führt den Job aus (z.B. Shell-Skripte, Docker-Container, etc.).
Sendet Logs und Status zurück.

## Features

- **YAML-basierte Job-Definitionen:** Jobs können einfach und transparent in YAML beschrieben werden.
- **Modulare Executor-Architektur:** Unterstützt verschiedene Ausführungsumgebungen (z.B. Shell, Docker) und ist erweiterbar.
- **Unabhängiges Logging:** Logs werden separat geschrieben und können einfach verfolgt werden.
- **Artefakt-Handling:** Outputs und Artefakte der Jobs werden gesammelt und verwaltet.
- **Callback-Mechanismus:** Ergebnisse und Statusupdates können an beliebige Endpunkte gesendet werden.
- **Daemon/Service-Betrieb:** Für kontinuierlichen Betrieb und Job-Management, ideal für Automatisierung.

## MVP Roadmap

1. Basis Runner, der Jobs aus YAML-Dateien laden und ausführen kann  
2. Modularer Executor für Shell- und Docker-Jobs  
3. Logging & Artefaktmanagement  
4. Callback-API für Statusupdates  
5. Service/Daemon-Modus zur automatischen Job-Verarbeitung  
6. Erweiterbare Executor-Plugins  
7. API-First-Architektur zur Backend-Anbindung (Backend bleibt frei wählbar)  

## Warum dieses Projekt?

Bestehende Tools sind oft stark auf bestimmte Anwendungsfälle (z.B. DevOps) oder Plattformen zugeschnitten. Dieser Runner bietet maximale Flexibilität und Unabhängigkeit, um beliebige Prozesse automatisiert und individuell steuerbar zu machen – offen, modular und erweiterbar.

## Einstieg

- Job-Definitionen in YAML erstellen  
- Runner starten und Jobs ausführen lassen  
- Logs und Artefakte im definierten Verzeichnis überprüfen  
- Callbacks nutzen, um Backend oder weitere Systeme zu benachrichtigen  


## Mitmachen & Feedback

Modular Runner ist Open Source und lebt von der Community!  
Jede Idee, Bug-Report oder Pull Request ist willkommen.

---

## Lizenz

MIT License

---

**Mach mit und gestalte die Zukunft der Hosting-Automatisierung mit Modular Runner!**

---




```bash
  docker run --rm \
    -v $(pwd)/tests:/app/tests \
    -v /var/run/docker.sock:/var/run/docker.sock \
    MASYONY/runner:latest run /app/tests/job-custom.yaml
```