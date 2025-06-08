# alarmego
## Functionality

* **Alarm Management:**
    * **Add Alarms:** Schedule recurring alarms with `alarmego add <interval> "<message>"`.
    * **Add One-Time Alarms:** Set single-trigger alarms using `alarmego addo <interval> "<message>"`.
    * **Remove Alarms:** Delete alarms by providing a partial message, utilizing Levenshtein distance for fuzzy matching (`alarmego remove "<partial_message>"`).
    * **Persistence:** Alarms are stored in `alarms.txt` and loaded upon startup.
* **Scheduling & Notifications:**
    * Alarms run concurrently as goroutines.
    * Desktop notifications are provided for Windows systems via PowerShell.

## Usage

1.  **Build:** `go build -o alarmego .`
2.  **Run:** `./alarmego` to start the alarm system.
3.  **Commands:**
    * `./alarmego add 1h "Take a break"`
    * `./alarmego addo 30m "Submit report"`
    * `./alarmego remove "break"`

## Requirements

* Go 1.22+
* Windows (for desktop notifications)
