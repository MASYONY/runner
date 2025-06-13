# Modular Job Runner

Ein flexibler, modularer Job Runner zur Ausführung beliebiger Prozesse wie Hosting-Provisionierung, Rechnungsprozesse, Deployment oder individuelle Workflows. Der Runner ist als eigenständiger Service konzipiert, der Jobs über konfigurierbare YAML-Dateien entgegennimmt, ausführt und die Ergebnisse inklusive Logs und Artefakten verwaltet.

## Vision

Der Runner soll als **unabhängiger, modularer Service** arbeiten, der über eine API angesteuert werden kann. Dadurch ist er flexibel in verschiedenste Backend-Systeme integrierbar und ermöglicht eine individuelle Steuerung und Erweiterung von Abläufen – von Provisionierung bis Abrechnung – ohne monolithisches System.

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


